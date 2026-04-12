package main

import (
	"os"
	"path/filepath"
)

func resolveWorkspace() string {
	if p := os.Getenv("TASKMANAGER_WORKSPACE"); p != "" {
		return p
	}
	if st, err := os.Stat("./backend"); err == nil && st.IsDir() {
		return filepath.FromSlash("./backend")
	}
	return "."
}
