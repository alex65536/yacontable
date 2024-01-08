package internal

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"math"
	"net/http"

	"go.uber.org/zap"
)

type Presenter struct {
	k      *Keeper
	ctx    context.Context
	logger *zap.Logger
	t      *template.Template
	conf   *Config
}

func getScoreColor(score float64) string {
	if score < 0.0 {
		score = 0.0
	}
	if score > 1.0 {
		score = 1.0
	}
	r := 0.0
	g := 0.0
	b := 0.0
	if score < 0.5 {
		d := score * 2.0
		r = 0.75 + 0.25*d
		g = 0.5 * d
		b = 0.0
	} else {
		d := (score - 0.5) * 2.0
		r = 1.0 - d
		g = 0.5
		b = 0
	}
	return fmt.Sprintf("#%02x%02x%02x", int(math.Round(r*255.0)), int(math.Round(g*255.0)), int(math.Round(b*255.0)))
}

func NewPresenter(ctx context.Context, logger *zap.Logger, k *Keeper, conf *Config) (*Presenter, error) {
	funcMap := template.FuncMap{
		"inc": func(i int) int {
			return i + 1
		},
		"supportsColor": func() bool {
			return conf.MaxScorePerTask != nil
		},
		"supportsFullScores": func() bool {
			return conf.MaxScorePerTask != nil
		},
		"calcColor": func(count int, score float64) string {
			return getScoreColor(score / (*conf.MaxScorePerTask * float64(count)))
		},
	}
	t, err := template.New("standings").Funcs(funcMap).ParseFiles("./data/standings.html")
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}
	return &Presenter{
		k:      k,
		ctx:    ctx,
		logger: logger,
		t:      t,
		conf:   conf,
	}, nil
}

func (p *Presenter) calcNumFullScores(st *Standings) []int {
	if p.conf.MaxScorePerTask == nil {
		return nil
	}
	res := make([]int, len(st.Header.Tasks))
	for _, pp := range st.Participants {
		for i, t := range pp.Tasks {
			if t.Score == *p.conf.MaxScorePerTask {
				res[i]++
			}
		}
	}
	return res
}

func (p *Presenter) doBuildTemplate(filter string) ([]byte, error) {
	type state struct {
		Filter     string
		Standings  *Standings
		FullScores []int
	}

	st, err := p.k.Get(p.ctx, p.logger)
	if err != nil {
		return nil, fmt.Errorf("getting statements: %w", err)
	}
	if filter != "" {
		st.FilterPrefix(filter)
	}
	var b bytes.Buffer
	err = p.t.ExecuteTemplate(&b, "standings.html", &state{
		Filter:     filter,
		Standings:  st,
		FullScores: p.calcNumFullScores(st),
	})
	if err != nil {
		return nil, fmt.Errorf("building template: %w", err)
	}
	return b.Bytes(), nil
}

func (p *Presenter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	p.logger.Info("get", zap.String("uri", req.RequestURI), zap.String("addr", req.RemoteAddr))
	if req.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = io.WriteString(w, "use GET method")
		return
	}
	query := req.URL.Query()
	filter := query.Get("filter")
	b, err := p.doBuildTemplate(filter)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		p.logger.Error("error serving request", zap.Error(err))
		_, _ = io.WriteString(w, "got error: "+err.Error())
		return
	}
	_, _ = w.Write(b)
}
