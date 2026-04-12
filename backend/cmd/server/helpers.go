package main

import (
	"os"
	"path/filepath"
)

func resolveDBPath() string {
	if p := os.Getenv("TASKMANAGER_DB_PATH"); p != "" {
		return p
	}
	if p := os.Getenv("DB_PATH"); p != "" {
		return p
	}
	if st, err := os.Stat("./cmd/server"); err == nil && st.IsDir() {
		return filepath.FromSlash("./tasks.db")
	}
	if st, err := os.Stat("./backend"); err == nil && st.IsDir() {
		return filepath.FromSlash("./backend/tasks.db")
	}
	return filepath.FromSlash("./tasks.db")
}

func resolveWorkspace() string {
	if p := os.Getenv("TASKMANAGER_WORKSPACE"); p != "" {
		return p
	}
	if st, err := os.Stat("./backend"); err == nil && st.IsDir() {
		return filepath.FromSlash("./backend")
	}
	return "."
}
