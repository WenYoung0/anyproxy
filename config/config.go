package config

import (
	"io"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Listen string `yaml:"listen,omitempty"`
	Port   uint16 `yaml:"port,omitempty"`

	Allowance []string `yaml:"allowance,omitempty"`

	Attempt     uint16 `yaml:"attempt,omitempty"`
	RateLimiter string `yaml:"rate-limit,omitempty"`
}

func Default() Config {
	return Config{}
}

func Parse(r io.Reader) (Config, error) {
	decoder := yaml.NewDecoder(r, yaml.DisallowUnknownField())
	c := Default()
	err := decoder.Decode(&c)
	if err != nil {
		return Config{}, err
	}
	return c, nil
}
