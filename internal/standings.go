package internal

import (
	"fmt"
	"regexp"
	"slices"

	"github.com/alex65536/yacontable/pkg/goutil"
	"go.uber.org/zap"
)

type ParticipantCell struct {
	Score float64 `json:"score"`
}

type TaskHeader struct {
	Name  string `json:"name"`
	Title string `json:"title"`
}

type Header struct {
	Tasks []TaskHeader `json:"tasks"`
}

type Participant struct {
	Login string            `json:"login"`
	Name  string            `json:"name"`
	Tasks []ParticipantCell `json:"tasks"`
	Total float64           `json:"total"`
}

type Standings struct {
	Tag          string        `json:"tag"`
	Header       Header        `json:"header"`
	Participants []Participant `json:"participants"`
}

func (s *Standings) ValidateAndFix() error {
	for i := range s.Participants {
		p := &s.Participants[i]
		if len(p.Tasks) != len(s.Header.Tasks) {
			return fmt.Errorf("participant %v:%v has %v tasks, but header has %v", i, p.Login, len(p.Tasks), len(s.Header.Tasks))
		}
		total := 0.0
		for _, v := range p.Tasks {
			total += v.Score
		}
		p.Total = total
	}
	s.sort()
	return nil
}

func (s *Standings) sort() {
	slices.SortFunc(s.Participants, func(a, b Participant) int {
		if a.Total > b.Total {
			return -1
		}
		if a.Total < b.Total {
			return 1
		}
		if a.Login < b.Login {
			return -1
		}
		if a.Login > b.Login {
			return 1
		}
		return 0
	})
}

func (s *Standings) Filter(loginRegex string) error {
	re, err := regexp.Compile(loginRegex)
	if err != nil {
		return fmt.Errorf("compiling regex: %w", err)
	}
	s.Participants = goutil.Filter(s.Participants, func(p Participant) bool {
		return re.MatchString(p.Login)
	})
	s.sort()
	return nil
}

func MergeStandings(logger *zap.Logger, sts ...*Standings) (*Standings, error) {
	type pinfo struct {
		used bool
		p    Participant
	}

	participants := make(map[string]*pinfo)
	for _, s := range sts {
		for _, p := range s.Participants {
			if val, ok := participants[p.Login]; ok {
				if val.p.Name != p.Name {
					logger.Warn("name mismatch for participant with the same login", zap.String("login", val.p.Login), zap.String("name1", val.p.Name), zap.String("name2", val.p.Name))
				}
			} else {
				participants[p.Login] = &pinfo{
					p: Participant{
						Login: p.Login,
						Name:  p.Name,
					},
				}
			}

		}
	}

	res := Standings{}
	for i, s := range sts {
		for _, p := range participants {
			p.used = false
		}
		for _, t := range s.Header.Tasks {
			tt := t
			if s.Tag != "" {
				tt.Title = fmt.Sprintf("%v-%v", s.Tag, tt.Title)
			}
			res.Header.Tasks = append(res.Header.Tasks, tt)
		}
		for _, p := range s.Participants {
			pp := participants[p.Login]
			if len(p.Tasks) != len(s.Header.Tasks) {
				logger.Error("number of tasks for participant mismatch, expect broken result", zap.Int("standings_id", i), zap.String("login", p.Login))
			}
			pp.p.Tasks = append(pp.p.Tasks, p.Tasks...)
			pp.used = true
		}
		for _, p := range participants {
			if !p.used {
				p.p.Tasks = append(p.p.Tasks, make([]ParticipantCell, len(s.Header.Tasks))...)
			}
		}
	}

	for _, p := range participants {
		res.Participants = append(res.Participants, p.p)
	}

	err := res.ValidateAndFix()
	if err != nil {
		return nil, fmt.Errorf("resulting standings are invalid: %w", err)
	}
	return &res, nil
}
