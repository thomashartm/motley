// Package config loads motley's configuration over its defaults.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Config contains the root paths used by motley.
type Config struct {
	Schema        int      `toml:"schema"`
	ReposRoot     string   `toml:"repos_root"`
	ReposRoots    []string `toml:"repos_roots"`
	WorktreesRoot string   `toml:"worktrees_root"`
	MonitorBell   bool     `toml:"monitor_bell"`
}

// Load creates the config once, defaults missing settings, and expands ~/.
func Load() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("find home directory: %w", err)
	}
	path, err := Ensure()
	if err != nil {
		return Config{}, err
	}
	cfg := Config{Schema: 1, ReposRoot: "~/projects", WorktreesRoot: "~/worktrees"}
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	if cfg.Schema != 1 {
		return Config{}, fmt.Errorf("config %s: unsupported schema %d (supported: 1)", path, cfg.Schema)
	}
	if strings.TrimSpace(cfg.WorktreesRoot) == "" {
		return Config{}, fmt.Errorf("config %s: worktrees_root must not be empty", path)
	}
	roots := cfg.RepositoryRoots()
	if len(roots) == 0 {
		return Config{}, fmt.Errorf("config %s: repos_roots must not be empty", path)
	}
	cfg.ReposRoots = nil
	seen := map[string]bool{}
	for _, root := range roots {
		if strings.TrimSpace(root) == "" {
			return Config{}, fmt.Errorf("config %s: repository roots must not be empty", path)
		}
		root = filepath.Clean(expandHome(root, home))
		if !seen[root] {
			cfg.ReposRoots = append(cfg.ReposRoots, root)
			seen[root] = true
		}
	}
	cfg.ReposRoot = cfg.ReposRoots[0]
	cfg.WorktreesRoot = expandHome(cfg.WorktreesRoot, home)
	return cfg, nil
}

// RepositoryRoots prefers the plural setting while supporting existing configs
// and callers that still supply the original single root.
func (c Config) RepositoryRoots() []string {
	if c.ReposRoots != nil {
		return c.ReposRoots
	}
	return []string{c.ReposRoot}
}

func expandHome(path, home string) string {
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}
