package internal

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type Keeper struct {
	conf  *Config
	api   *Api
	teams *TeamAssigner

	mu        sync.RWMutex
	st        *Standings
	err       error
	fetched   bool
	fetchTime time.Time
}

func NewKeeper(conf *Config, api *Api) (*Keeper, error) {
	teams, err := NewTeamAssigner(conf)
	if err != nil {
		return nil, fmt.Errorf("creating team assigner: %w", err)
	}
	return &Keeper{
		conf:  conf,
		api:   api,
		teams: teams,
	}, nil
}

func (k *Keeper) Get(ctx context.Context, logger *zap.Logger) (*Standings, error) {
	st, err, ok := k.tryGetSimple()
	if ok {
		return st, err
	}
	return k.doGetHeavy(ctx, logger)
}

func (k *Keeper) needsFetchUnlocked() bool {
	if !k.fetched {
		return true
	}
	var interval time.Duration
	if k.err == nil {
		interval = k.conf.RefreshDuration
	} else {
		interval = k.conf.ErrorRefreshDuration
	}
	if time.Now().After(k.fetchTime.Add(interval)) {
		return true
	}
	return false
}

func (k *Keeper) tryGetSimple() (*Standings, error, bool) {
	k.mu.RLock()
	defer k.mu.RUnlock()
	if !k.needsFetchUnlocked() {
		return k.st, k.err, true
	}
	return nil, nil, false
}

func (k *Keeper) doGetHeavy(ctx context.Context, logger *zap.Logger) (*Standings, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	if !k.needsFetchUnlocked() {
		return k.st, k.err
	}

	res := make([]*Standings, len(k.conf.Contests))

	logger.Info("refreshing standings", zap.Time("fetch_time", k.fetchTime))
	g, _ := errgroup.WithContext(ctx)
	for i := range k.conf.Contests {
		i := i
		ct := k.conf.Contests[i]
		g.Go(func() error {
			st, err := func() (*Standings, error) {
				st, err := k.api.FetchStandings(ct)
				if err != nil {
					return nil, err
				}
				for i := range st.Participants {
					st.Participants[i] = k.teams.AssignTeam(st.Participants[i])
				}
				st, err = st.FilterRegex(k.conf.LoginWhitelistRegex, FilterModeWhitelist)
				st, err = st.FilterRegex(k.conf.LoginBlacklistRegex, FilterModeBlacklist)
				if err != nil {
					return nil, err
				}
				return st, nil
			}()
			res[i] = st
			if err != nil {
				logger.Info("got error while refreshing standings", zap.Error(err))
			}
			return err
		})
	}
	err := g.Wait()
	var st *Standings
	if err == nil {
		st, err = MergeStandings(logger, res...)
	}
	k.st = st
	k.err = err
	k.fetched = true
	k.fetchTime = time.Now()
	return st, err
}
