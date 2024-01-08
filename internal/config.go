package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"golang.org/x/oauth2"
)

type Contest struct {
	ID  int    `json:"id"`
	Tag string `json:"tag"`
}

type Config struct {
	ListenAddr           string        `json:"listen_addr"`
	Domain               string        `json:"domain"`
	Contests             []Contest     `json:"contests"`
	RefreshDuration      time.Duration `json:"refresh_duration"`
	ErrorRefreshDuration time.Duration `json:"error_refresh_duration"`
	StandingsForJudge    bool          `json:"standings_for_judge"`
	PageSize             int           `json:"page_size"`
	LoginWhitelistRegex   string `json:"login_whitelist_regex"`
	MaxScorePerTask *float64 `json:"max_score_per_task"`
}

func (c *Config) FillDefaults() {
	if c.ListenAddr == "" {
		c.ListenAddr = "0.0.0.0:4242"
	}
	if c.Domain == "" {
		c.Domain = "http://localhost:4242"
	}
	if c.RefreshDuration == 0 {
		c.RefreshDuration = 60 * time.Second
	}
	if c.ErrorRefreshDuration == 0 {
		c.ErrorRefreshDuration = 1 * time.Second
	}
	if c.PageSize == 0 {
		c.PageSize = 10000
	}
	if c.LoginWhitelistRegex == "" {
		c.LoginWhitelistRegex = ".*"
	}
}

type StaticSecrets struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type DynamicSecrets struct {
	Token *oauth2.Token `json:"token"`
}

func unmarshalFromFile(name string, value any) error {
	f, err := os.Open(name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("reading file: %w", err)
	}
	defer f.Close()
	d := json.NewDecoder(f)
	err = d.Decode(value)
	if err != nil {
		return fmt.Errorf("unmarshalling json: %w", err)
	}
	return nil
}

func LoadConfig() (*Config, error) {
	var c Config
	err := unmarshalFromFile("config.json", &c)
	if err != nil {
		return nil, err
	}
	c.FillDefaults()
	return &c, nil
}

func LoadStaticSecrets() (*StaticSecrets, error) {
	var s StaticSecrets
	err := unmarshalFromFile("secrets/static.json", &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func LoadDynamicSecrets() (*DynamicSecrets, error) {
	var s DynamicSecrets
	err := unmarshalFromFile("secrets/dynamic.json", &s)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func StoreDynamicSecrets(s *DynamicSecrets) error {
	f, err := os.Create("secrets/dynamic.json")
	if err != nil {
		return fmt.Errorf("storing dynamic secrets: %w", err)
	}
	defer f.Close()
	e := json.NewEncoder(f)
	err = e.Encode(s)
	if err != nil {
		return fmt.Errorf("encoding json: %w", err)
	}
	return nil
}
