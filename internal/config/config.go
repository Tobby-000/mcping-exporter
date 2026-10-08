package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const maxLimit = 10000

type Target struct {
	Name string `yaml:"name"`
	Addr string `yaml:"addr"`
}

type ProbeConfig struct {
	Interval int `yaml:"interval"`
	Timeout  int `yaml:"timeout"`
	Limit    int `yaml:"limit"`
}
type Config struct {
	Probe   ProbeConfig `yaml:"probe"`
	Targets []Target    `yaml:"targets"`
}

func Load(path string) (*Config, time.Time, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("read config %s: %w", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("stat config %s: %w", path, err)
	}
	cfg := Config{
		Probe: ProbeConfig{
			Interval: 15,
			Timeout:  5,
			Limit:    20,
		},
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, time.Time{}, fmt.Errorf("decode yaml %s: %w", path, err)
	}
	if err := cfg.validate(); err != nil {
		return nil, time.Time{}, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, info.ModTime(), nil
}

func (c *Config) validate() error {
	// probe check
	if c.Probe.Interval <= 0 {
		return fmt.Errorf("probe.interval must be > 0")
	}
	if c.Probe.Limit <= 0 {
		return fmt.Errorf("probe.limit must be > 0")
	}
	if c.Probe.Timeout <= 0 {
		return fmt.Errorf("probe.timeout must be > 0")
	}
	if c.Probe.Interval <= c.Probe.Timeout {
		return fmt.Errorf("probe.timeout (%d) must be < probe.interval(%d)", c.Probe.Timeout, c.Probe.Interval)
	}
	if c.Probe.Limit > maxLimit {
		return fmt.Errorf("probe.limit is too large (%d),must be <= %d", c.Probe.Limit, maxLimit)
	}
	// targets check
	if err := validateTargets(c.Targets); err != nil {
		return fmt.Errorf("targets: %w", err)
	}
	return nil
}
func (c *Config) Warns() []string {
	var w []string
	if c.Probe.Limit > 100 {
		w = append(w, fmt.Sprintf(
			"probe.limit (%d) is very high, consider reducing it to <= 100",
			c.Probe.Limit,
		))
	}
	if len(c.Targets) > 0 {
		capacity := c.Probe.Limit * (c.Probe.Interval / c.Probe.Timeout)
		if len(c.Targets) > capacity {
			w = append(w, fmt.Sprintf(
				"target count (%d) exceeds probe capacity (%d = limit %d x interval %ds / timeout %ds), rounds will be delayed",
				len(c.Targets), capacity, c.Probe.Limit, c.Probe.Interval, c.Probe.Timeout,
			))
		}
	}
	return w
}
func validateTargets(targets []Target) error {
	seen := make(map[string]struct{}, len(targets))
	for i, t := range targets {
		if t.Addr == "" {
			return fmt.Errorf("targets[%d] (%s): addr is empty", i, t.Name)
		}
		if t.Name == "" {
			return fmt.Errorf("targets[%d]: name is empty", i)
		}
		if _, ok := seen[t.Name]; ok {
			return fmt.Errorf("duplicate target name: %s", t.Name)
		}
		seen[t.Name] = struct{}{}
	}
	return nil
}
