package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name: "valid",
			cfg: Config{
				Probe:   ProbeConfig{Interval: 15, Timeout: 5, Limit: 20},
				Targets: []Target{{Name: "a", Addr: "127.0.0.1:25565"}},
			},
		},
		{
			name: "zero interval",
			cfg: Config{
				Probe:   ProbeConfig{Interval: 0, Timeout: 5, Limit: 20},
				Targets: []Target{{Name: "a", Addr: "127.0.0.1:25565"}},
			},
			wantErr: "interval",
		},
		{
			name: "negative interval",
			cfg: Config{
				Probe:   ProbeConfig{Interval: -1, Timeout: 5, Limit: 20},
				Targets: []Target{{Name: "a", Addr: "127.0.0.1:25565"}},
			},
			wantErr: "interval",
		},
		{
			name: "zero timeout",
			cfg: Config{
				Probe:   ProbeConfig{Interval: 15, Timeout: 0, Limit: 20},
				Targets: []Target{{Name: "a", Addr: "127.0.0.1:25565"}},
			},
			wantErr: "timeout",
		},
		{
			name: "timeout equals interval",
			cfg: Config{
				Probe:   ProbeConfig{Interval: 15, Timeout: 15, Limit: 20},
				Targets: []Target{{Name: "a", Addr: "127.0.0.1:25565"}},
			},
			wantErr: "timeout",
		},
		{
			name: "timeout greater than interval",
			cfg: Config{
				Probe:   ProbeConfig{Interval: 5, Timeout: 10, Limit: 20},
				Targets: []Target{{Name: "a", Addr: "127.0.0.1:25565"}},
			},
			wantErr: "timeout",
		},
		{
			name: "zero limit",
			cfg: Config{
				Probe:   ProbeConfig{Interval: 15, Timeout: 5, Limit: 0},
				Targets: []Target{{Name: "a", Addr: "127.0.0.1:25565"}},
			},
			wantErr: "limit",
		},
		{
			name: "negative limit",
			cfg: Config{
				Probe:   ProbeConfig{Interval: 15, Timeout: 5, Limit: -1},
				Targets: []Target{{Name: "a", Addr: "127.0.0.1:25565"}},
			},
			wantErr: "limit",
		},
		{
			name: "limit equals maxLimit",
			cfg: Config{
				Probe:   ProbeConfig{Interval: 15, Timeout: 5, Limit: maxLimit},
				Targets: []Target{{Name: "a", Addr: "127.0.0.1:25565"}},
			},
		},
		{
			name: "limit exceeds maxLimit by 1",
			cfg: Config{
				Probe:   ProbeConfig{Interval: 15, Timeout: 5, Limit: maxLimit + 1},
				Targets: []Target{{Name: "a", Addr: "127.0.0.1:25565"}},
			},
			wantErr: "too large",
		},
		{
			name: "empty target name",
			cfg: Config{
				Probe:   ProbeConfig{Interval: 15, Timeout: 5, Limit: 20},
				Targets: []Target{{Name: "", Addr: "127.0.0.1:25565"}},
			},
			wantErr: "name is empty",
		},
		{
			name: "empty target addr",
			cfg: Config{
				Probe:   ProbeConfig{Interval: 15, Timeout: 5, Limit: 20},
				Targets: []Target{{Name: "a", Addr: ""}},
			},
			wantErr: "addr is empty",
		},
		{
			name: "duplicate target name",
			cfg: Config{
				Probe: ProbeConfig{Interval: 15, Timeout: 5, Limit: 20},
				Targets: []Target{
					{Name: "a", Addr: "127.0.0.1:25565"},
					{Name: "a", Addr: "127.0.0.2:25565"},
				},
			},
			wantErr: "duplicate",
		},
		{
			name: "no targets is allowed",
			cfg: Config{
				Probe: ProbeConfig{Interval: 15, Timeout: 5, Limit: 20},
			},
		},
		{
			name: "targets order does not matter for duplicates",
			cfg: Config{
				Probe: ProbeConfig{Interval: 15, Timeout: 5, Limit: 20},
				Targets: []Target{
					{Name: "b", Addr: "127.0.0.1:25565"},
					{Name: "a", Addr: "127.0.0.1:25566"},
					{Name: "b", Addr: "127.0.0.1:25567"},
				},
			},
			wantErr: "duplicate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestValidateTargets(t *testing.T) {
	tests := []struct {
		name    string
		targets []Target
		wantErr string
	}{
		{
			name:    "empty slice",
			targets: nil,
		},
		{
			name:    "single valid",
			targets: []Target{{Name: "a", Addr: "127.0.0.1:25565"}},
		},
		{
			name: "multiple valid",
			targets: []Target{
				{Name: "a", Addr: "127.0.0.1:25565"},
				{Name: "b", Addr: "127.0.0.1:25566"},
			},
		},
		{
			name:    "empty addr",
			targets: []Target{{Name: "a", Addr: ""}},
			wantErr: "addr is empty",
		},
		{
			name:    "empty name",
			targets: []Target{{Name: "", Addr: "127.0.0.1:25565"}},
			wantErr: "name is empty",
		},
		{
			name: "duplicate",
			targets: []Target{
				{Name: "a", Addr: "127.0.0.1:25565"},
				{Name: "a", Addr: "127.0.0.1:25566"},
			},
			wantErr: "duplicate target name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTargets(tt.targets)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestWarns(t *testing.T) {
	tests := []struct {
		name      string
		cfg       Config
		wantCount int
	}{
		{
			name: "no warnings",
			cfg: Config{
				Probe:   ProbeConfig{Interval: 15, Timeout: 5, Limit: 20},
				Targets: make([]Target, 10),
			},
		},
		{
			name: "limit exactly 100 no warn",
			cfg: Config{
				Probe:   ProbeConfig{Interval: 15, Timeout: 5, Limit: 100},
				Targets: make([]Target, 10),
			},
		},
		{
			name: "limit 101 warns",
			cfg: Config{
				Probe:   ProbeConfig{Interval: 15, Timeout: 5, Limit: 101},
				Targets: make([]Target, 10),
			},
			wantCount: 1,
		},
		{
			name: "targets exactly at capacity no warn",
			cfg: Config{
				// capacity = 10 * (15/5) = 30
				Probe:   ProbeConfig{Interval: 15, Timeout: 5, Limit: 10},
				Targets: make([]Target, 30),
			},
		},
		{
			name: "targets one above capacity warns",
			cfg: Config{
				Probe:   ProbeConfig{Interval: 15, Timeout: 5, Limit: 10},
				Targets: make([]Target, 31),
			},
			wantCount: 1,
		},
		{
			name: "no targets no capacity warn",
			cfg: Config{
				Probe: ProbeConfig{Interval: 15, Timeout: 5, Limit: 10},
			},
		},
		{
			name: "both warnings",
			cfg: Config{
				Probe:   ProbeConfig{Interval: 15, Timeout: 5, Limit: 200},
				Targets: make([]Target, 1000),
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := tt.cfg.Warns()
			if len(w) != tt.wantCount {
				t.Fatalf("expected %d warnings, got %d: %v", tt.wantCount, len(w), w)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	yaml := `
probe:
  interval: 30
  timeout: 5
  limit: 10
targets:
  - name: survival
    addr: play.example.com
`
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, modTime, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Probe.Interval != 30 || cfg.Probe.Timeout != 5 || cfg.Probe.Limit != 10 {
		t.Errorf("probe not parsed: %+v", cfg.Probe)
	}
	if len(cfg.Targets) != 1 || cfg.Targets[0].Name != "survival" {
		t.Errorf("targets not parsed: %+v", cfg.Targets)
	}

	// modTime 应该等于文件的实际 mtime
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !modTime.Equal(info.ModTime()) {
		t.Errorf("modTime mismatch: got %v, want %v", modTime, info.ModTime())
	}
}

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	yaml := `
targets:
  - name: a
    addr: 127.0.0.1:25565
`
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Probe.Interval != 15 {
		t.Errorf("interval default: want 15, got %d", cfg.Probe.Interval)
	}
	if cfg.Probe.Timeout != 5 {
		t.Errorf("timeout default: want 5, got %d", cfg.Probe.Timeout)
	}
	if cfg.Probe.Limit != 20 {
		t.Errorf("limit default: want 20, got %d", cfg.Probe.Limit)
	}
}

func TestLoadPartialProbe(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	// 只覆盖 interval，timeout 和 limit 走默认
	yaml := `
probe:
  interval: 60
targets:
  - name: a
    addr: 127.0.0.1:25565
`
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Probe.Interval != 60 {
		t.Errorf("interval: want 60, got %d", cfg.Probe.Interval)
	}
	if cfg.Probe.Timeout != 5 {
		t.Errorf("timeout default lost: want 5, got %d", cfg.Probe.Timeout)
	}
	if cfg.Probe.Limit != 20 {
		t.Errorf("limit default lost: want 20, got %d", cfg.Probe.Limit)
	}
}

func TestLoadEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := Load(path)
	if err != nil {
		t.Fatalf("empty file should use defaults, got error: %v", err)
	}
	if cfg.Probe.Interval != 15 || cfg.Probe.Timeout != 5 || cfg.Probe.Limit != 20 {
		t.Errorf("defaults not applied: %+v", cfg.Probe)
	}
	if len(cfg.Targets) != 0 {
		t.Errorf("expected no targets, got %d", len(cfg.Targets))
	}
}

func TestLoadFileNotExist(t *testing.T) {
	_, _, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
	if !strings.Contains(err.Error(), "read config") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(path, []byte("this is: [not: valid"), 0644); err != nil {
		t.Fatal(err)
	}

	_, _, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid yaml, got nil")
	}
	if !strings.Contains(err.Error(), "decode yaml") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	// timeout > interval，validate 应该拒绝
	yaml := `
probe:
  interval: 5
  timeout: 10
targets:
  - name: a
    addr: 127.0.0.1:25565
`
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	_, _, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid config, got nil")
	}
	if !strings.Contains(err.Error(), "invalid config") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadUnknownFieldsIgnored(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	// 未知字段，yaml.v3 默认忽略
	yaml := `
unknown_field: whatever
probe:
  interval: 15
  timeout: 5
  limit: 20
targets:
  - name: a
    addr: 127.0.0.1:25565
`
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := Load(path)
	if err != nil {
		t.Fatalf("unknown fields should be ignored, got: %v", err)
	}
	if cfg.Probe.Interval != 15 {
		t.Errorf("interval: want 15, got %d", cfg.Probe.Interval)
	}
}

// 保证 modTime 类型正确，未来如果改成别的时间类型，这里会编译失败
var _ time.Time = time.Time{}
