package internal

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/endpoints"

	"github.com/alex65536/yacontable/pkg/goutil"
)

type Api struct {
	client *http.Client
	conf   *Config
	logger *zap.Logger
}

func NewApi(logger *zap.Logger, ctx context.Context, conf *Config, s *StaticSecrets) (*Api, error) {
	d, err := LoadDynamicSecrets()
	if err != nil {
		return nil, fmt.Errorf("loading dynamic secrets: %w", err)
	}

	oauthConf := &oauth2.Config{
		ClientID:     s.ClientID,
		ClientSecret: s.ClientSecret,
		Endpoint:     endpoints.Yandex,
		Scopes:       []string{"contest:submit"},
		RedirectURL:  fmt.Sprintf("%v/authCallback", conf.Domain),
	}

	if d.Token == nil {
		d, err = fetchTokens(logger, ctx, conf, oauthConf, s)
		if err != nil {
			return nil, fmt.Errorf("requesting oauth2 tokens: %w", err)
		}
		err = StoreDynamicSecrets(d)
		if err != nil {
			return nil, fmt.Errorf("storing dynamic secrets: %w", err)
		}
	}

	return &Api{
		client: oauthConf.Client(ctx, d.Token),
		conf:   conf,
		logger: logger,
	}, nil
}

func parseScore(src string) (float64, error) {
	if src == "" {
		return 0.0, nil
	}
	if !('0' <= src[0] && src[0] <= '9') {
		return 0.0, fmt.Errorf("score must start with digit")
	}
	src = strings.Replace(src, ",", ".", -1)
	return strconv.ParseFloat(src, 64)
}

func (a *Api) FetchStandings(contest Contest) (*Standings, error) {
	type title struct {
		Name  string `json:"name"`
		Title string `json:"title"`
	}

	type partInfo struct {
		Login string `json:"login"`
		Name  string `json:"name"`
	}

	type problemResult struct {
		Score string `json:"score"`
	}

	type row struct {
		ParticipantInfo partInfo        `json:"participantInfo"`
		ProblemResults  []problemResult `json:"problemResults"`
	}

	type standings struct {
		Titles []title `json:"titles"`
		Rows   []row   `json:"rows"`
	}

	v := url.Values{}
	v.Add("forJudge", fmt.Sprintf("%v", a.conf.StandingsForJudge))
	v.Add("page", "1")
	v.Add("pageSize", fmt.Sprintf("%v", a.conf.PageSize))

	rsp, err := a.client.Get(fmt.Sprintf("https://api.contest.yandex.net/api/public/v2/contests/%v/standings?%v", contest.ID, v.Encode()))
	defer func() {
		_, _ = io.Copy(io.Discard, rsp.Body)
		_ = rsp.Body.Close()
	}()
	if rsp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(rsp.Body)
		a.logger.Error("non-ok response body", zap.String("data", string(data)))
		return nil, fmt.Errorf("got non-ok status from contest API: %v %v", rsp.StatusCode, rsp.Status)
	}

	d := json.NewDecoder(rsp.Body)
	var st standings
	err = d.Decode(&st)
	if err != nil {
		return nil, fmt.Errorf("decoding json standings: %w", err)
	}

	participants, err := goutil.MapWithErr(st.Rows, func(r row) (Participant, error) {
		tasks, err := goutil.MapWithErr(r.ProblemResults, func(p problemResult) (ParticipantCell, error) {
			score, err := parseScore(p.Score)
			if err != nil {
				return ParticipantCell{}, fmt.Errorf("decoding float score %q: %w", p.Score, err)
			}
			return ParticipantCell{
				Score: score,
			}, nil
		})
		if err != nil {
			return Participant{}, fmt.Errorf("decoding task results: %w", err)
		}
		return Participant{
			Login: r.ParticipantInfo.Login,
			Name:  r.ParticipantInfo.Name,
			Tasks: tasks,
		}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parsing participants: %w", err)
	}

	res := &Standings{
		Tag: contest.Tag,
		Header: Header{
			Tasks: goutil.Map(st.Titles, func(t title) TaskHeader {
				return TaskHeader{
					Name:  t.Name,
					Title: t.Title,
				}
			}),
		},
		Participants: participants,
	}

	err = res.ValidateAndFix()
	if err != nil {
		return nil, fmt.Errorf("validating standings: %w", err)
	}

	return res, nil
}

func fetchTokens(logger *zap.Logger, ctx context.Context, conf *Config, oauthConf *oauth2.Config, s *StaticSecrets) (*DynamicSecrets, error) {
	verifier := oauth2.GenerateVerifier()
	url := oauthConf.AuthCodeURL(callback.state, oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(verifier))
	logger.Info("generated auth code URL", zap.String("url", url))
	fmt.Printf("to authorize, please visit %v\n", url)
	code := <-callback.codes
	tok, err := oauthConf.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, fmt.Errorf("cannot exchange code for token: %w", err)
	}
	tok.TokenType = "OAuth" // HACK
	return &DynamicSecrets{
		Token: tok,
	}, nil
}

type authCallback struct {
	state string
	codes chan string
}

func genState() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	var b strings.Builder
	for i := 0; i < 24; i++ {
		pos, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			panic(fmt.Errorf("cannot generate state: %w", err))
		}
		_ = b.WriteByte(letters[pos.Int64()])
	}
	return b.String()
}

var callback *authCallback = &authCallback{
	state: genState(),
	codes: make(chan string, 1000),
}

func init() {
	http.HandleFunc("/authCallback", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			_, _ = io.WriteString(w, "use GET method")
			return
		}
		query := req.URL.Query()
		state := query.Get("state")
		if state == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, "no state")
			return
		}
		if subtle.ConstantTimeCompare([]byte(callback.state), []byte(state)) == 0 {
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, "bad state")
			return
		}
		code := query.Get("code")
		if code == "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, "no code")
			return
		}
		callback.codes <- code
	})
}
