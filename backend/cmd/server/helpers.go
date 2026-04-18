package main

import (
	"os"
	"path/filepath"

	"github.com/weibh/taskmanager/infrastructure/config"
)

func resolveWorkspace() string {
	if p := config.GetWorkspace(); p != "" {
		return p
	}
	if st, err := os.Stat("./backend"); err == nil && st.IsDir() {
		return filepath.FromSlash("./backend")
	}
	return "."
}
