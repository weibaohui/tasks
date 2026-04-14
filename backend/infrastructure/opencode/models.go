package opencode

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// ListModels 执行 `opencode models` 命令获取可用模型列表
func ListModels() ([]string, error) {
	path, err := exec.LookPath("opencode")
	if err != nil {
		return nil, &OpenCodeNotFoundError{err: err}
	}

	cmd := exec.Command(path, "models")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return nil, fmt.Errorf("opencode models failed: %w (stderr: %s)", err, stderrStr)
		}
		return nil, fmt.Errorf("opencode models failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	models := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			models = append(models, line)
		}
	}
	return models, nil
}
