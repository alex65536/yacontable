package internal

import (
	"fmt"
	"regexp"
)

type TeamAssigner struct {
	logins   map[string]int
	patterns [][]*regexp.Regexp
}

func NewTeamAssigner(conf *Config) (*TeamAssigner, error) {
	logins := make(map[string]int)
	patterns := make([][]*regexp.Regexp, len(conf.Teams))
	for i, team := range conf.Teams {
		for _, login := range team.Logins {
			logins[login] = i
		}
		for _, pattern := range team.Patterns {
			pat, err := regexp.Compile(pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid pattern %q: %w", pattern, err)
			}
			patterns[i] = append(patterns[i], pat)
		}
	}
	return &TeamAssigner{
		logins:   logins,
		patterns: patterns,
	}, nil
}

func (t *TeamAssigner) AssignTeam(p Participant) Participant {
	if teamID, ok := t.logins[p.Login]; ok {
		p.TeamID = teamID
		return p
	}
	for i, pats := range t.patterns {
		for _, pat := range pats {
			if pat.MatchString(p.Login) {
				p.TeamID = i
				return p
			}
		}
	}
	p.TeamID = -1
	return p
}
