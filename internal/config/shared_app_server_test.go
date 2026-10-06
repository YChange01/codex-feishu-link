package config

import (
	"path/filepath"
	"testing"
)

func TestLoadWrapperConfigSharedAppServer(t *testing.T) {
	for _, tt := range []struct {
		name      string
		persisted bool
		env       string
		want      bool
	}{
		{name: "standalone default"},
		{name: "persisted shared mode", persisted: true, want: true},
		{name: "environment enables shared mode", env: "true", want: true},
		{name: "environment disables persisted shared mode", persisted: true, env: "false"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "config.json")
			t.Setenv(UnifiedConfigEnvPath, configPath)
			t.Setenv(XDGConfigHomeEnv, t.TempDir())
			t.Setenv(SharedAppServerEnv, tt.env)
			cfg := DefaultAppConfig()
			cfg.Wrapper.SharedAppServer = tt.persisted
			if err := WriteAppConfig(configPath, cfg); err != nil {
				t.Fatalf("write persisted wrapper config: %v", err)
			}

			loaded, err := LoadWrapperConfig()
			if err != nil {
				t.Fatalf("load wrapper config: %v", err)
			}
			if loaded.ConfigPath != configPath {
				t.Fatalf("loaded config from %q, want isolated fixture %q", loaded.ConfigPath, configPath)
			}
			if loaded.SharedAppServer != tt.want {
				t.Fatalf("SharedAppServer = %t, want %t (persisted=%t, env=%q)", loaded.SharedAppServer, tt.want, tt.persisted, tt.env)
			}
		})
	}
}
