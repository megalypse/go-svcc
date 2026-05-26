package cfg

import (
	"os"
	"path"
	"sync"
)

var GetConfig = sync.OnceValues(loadCfg)

func loadCfg() (*Config, error) {
	dir := os.Getenv("SVCC_PATH")

	if dir == "" {
		home := os.Getenv("HOME")
		dir = path.Join(home, "svcc")
	}
	return &Config{
		Path: dir,
	}, nil
}
