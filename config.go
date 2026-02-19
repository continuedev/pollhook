package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Sources []Source `yaml:"sources"`
}

type Source struct {
	Name     string   `yaml:"name"`
	Command  string   `yaml:"command"`
	Interval Duration `yaml:"interval"`
	Items    string   `yaml:"items"`
	ID       string   `yaml:"id"`
	Webhook  Webhook  `yaml:"webhook"`
}

type Webhook struct {
	URL    string `yaml:"url"`
	Secret string `yaml:"secret"`
}

// Duration wraps time.Duration for YAML unmarshaling from strings like "5m".
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	d.Duration = dur
	return nil
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	// Expand environment variables in the config
	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if len(c.Sources) == 0 {
		return fmt.Errorf("config: no sources defined")
	}

	names := make(map[string]bool)
	for i, src := range c.Sources {
		if src.Name == "" {
			return fmt.Errorf("config: source[%d] missing name", i)
		}
		if names[src.Name] {
			return fmt.Errorf("config: duplicate source name %q", src.Name)
		}
		names[src.Name] = true

		if src.Command == "" {
			return fmt.Errorf("config: source %q missing command", src.Name)
		}
		if src.Interval.Duration < time.Second {
			return fmt.Errorf("config: source %q interval must be >= 1s", src.Name)
		}
		if src.Items == "" {
			return fmt.Errorf("config: source %q missing items path", src.Name)
		}
		if src.ID == "" {
			return fmt.Errorf("config: source %q missing id path", src.Name)
		}
		if src.Webhook.URL == "" {
			return fmt.Errorf("config: source %q missing webhook url", src.Name)
		}
	}
	return nil
}
