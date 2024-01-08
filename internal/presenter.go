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
	}, nil
}

func (p *Presenter) doBuildTemplate() ([]byte, error) {
	st, err := p.k.Get(p.ctx, p.logger)
	if err != nil {
		return nil, fmt.Errorf("getting statements: %w", err)
	}
	var b bytes.Buffer
	err = p.t.ExecuteTemplate(&b, "standings.html", st)
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
	b, err := p.doBuildTemplate()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		p.logger.Error("error serving request", zap.Error(err))
		_, _ = io.WriteString(w, "got error: "+err.Error())
		return
	}
	_, _ = w.Write(b)
}
