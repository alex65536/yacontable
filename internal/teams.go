package internal

import (
	"fmt"
	"regexp"
)

type teamPattern struct {
	teamID int
	pat    *regexp.Regexp
}

type TeamAssigner struct {
	logins   map[string]int
	patterns []teamPattern
}

func NewTeamAssigner(conf *Config) (*TeamAssigner, error) {
	logins := make(map[string]int)
	var patterns []teamPattern
	for i, team := range conf.Teams {
		for _, login := range team.Logins {
			logins[login] = i
		}
		for _, pattern := range team.Patterns {
			pat, err := regexp.Compile(pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid pattern %q: %w", pattern, err)
			}
			patterns = append(patterns, teamPattern{
				teamID: i,
				pat:    pat,
			})
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
	for _, pat := range t.patterns {
		if pat.pat.MatchString(p.Login) {
			p.TeamID = pat.teamID
			return p
		}
	}
	p.TeamID = -1
	return p
}
