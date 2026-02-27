package config

import (
	"fmt"
	"net/url"

	"github.com/caarlos0/env/v11"
)

type Scaleway struct {
	APIBaseURL     string `env:"IMPACT_SCW_API_BASE_URL" envDefault:"https://api.scaleway.com"`
	AccessKey      string `env:"SCW_ACCESS_KEY"`
	SecretKey      string `env:"SCW_SECRET_KEY"`
	OrganizationID string `env:"SCW_ORGANIZATION_ID"`
}

func LoadScalewayFromEnv() (Scaleway, error) {
	var cfg Scaleway
	if err := env.Parse(&cfg); err != nil {
		return Scaleway{}, fmt.Errorf("could not parse env config: %w", err)
	}

	baseURL, err := url.Parse(cfg.APIBaseURL)
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return Scaleway{}, fmt.Errorf("could not validate IMPACT_SCW_API_BASE_URL: must be a valid absolute URL")
	}

	if baseURL.Scheme != "https" {
		return Scaleway{}, fmt.Errorf("could not validate IMPACT_SCW_API_BASE_URL: https scheme is required")
	}
	return cfg, nil
}
