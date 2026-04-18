package http

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	infraConfig "github.com/weibh/taskmanager/infrastructure/config"
)

const (
	defaultTailLines   = 200
	maxTailLines       = 5000
	maxStreamReadBytes = 1024 * 1024
	streamPollInterval = 800 * time.Millisecond
)

// SystemLogLine 表示日志中的一行数据。
type SystemLogLine struct {
	Line int64  `json:"line"`
	Text string `json:"text"`
}

// SystemLogTailResponse 表示日志尾部读取结果。
type SystemLogTailResponse struct {
	Path      string          `json:"path"`
	Keyword   string          `json:"keyword"`
	Total     int64           `json:"total"`
	Truncated bool            `json:"truncated"`
	Lines     []SystemLogLine `json:"lines"`
}

// SystemLogStreamEvent 表示 SSE 推送事件。
type SystemLogStreamEvent struct {
	Type      string          `json:"type"`
	Path      string          `json:"path,omitempty"`
	Keyword   string          `json:"keyword,omitempty"`
	Truncated bool            `json:"truncated,omitempty"`
	Total     int64           `json:"total,omitempty"`
	Lines     []SystemLogLine `json:"lines,omitempty"`
	Message   string          `json:"message,omitempty"`
}

// SystemLogHandler 提供系统日志读取和流式推送能力。
type SystemLogHandler struct{}

// NewSystemLogHandler 创建系统日志处理器。
func NewSystemLogHandler() *SystemLogHandler {
	return &SystemLogHandler{}
}

// GetConfig 返回当前日志文件配置路径。
func (h *SystemLogHandler) GetConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"path": infraConfig.GetServerLogPath(),
	})
}

// GetTail 按行数读取日志尾部内容，支持关键字过滤。
func (h *SystemLogHandler) GetTail(c *gin.Context) {
	logPath := infraConfig.GetServerLogPath()
	lines := parseTailLines(c.Query("lines"))
	keyword := strings.TrimSpace(c.Query("keyword"))

	result, err := readTailLines(logPath, lines, keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, SystemLogTailResponse{
		Path:      logPath,
		Keyword:   keyword,
		Total:     result.Total,
		Truncated: result.Truncated,
		Lines:     result.Lines,
	})
}

// Stream 通过 SSE 持续推送日志增量内容，支持关键字过滤。
func (h *SystemLogHandler) Stream(c *gin.Context) {
	logPath := infraConfig.GetServerLogPath()
	lines := parseTailLines(c.Query("lines"))
	keyword := strings.TrimSpace(c.Query("keyword"))

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: "当前连接不支持流式输出"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	initial, err := readTailLines(logPath, lines, keyword)
	if err != nil {
		_ = writeSSE(c.Writer, SystemLogStreamEvent{
			Type:    "error",
			Message: err.Error(),
		})
		flusher.Flush()
		return
	}

	offset, err := getFileSize(logPath)
	if err != nil {
		_ = writeSSE(c.Writer, SystemLogStreamEvent{
			Type:    "error",
			Message: err.Error(),
		})
		flusher.Flush()
		return
	}

	lineNo := initial.Total
	_ = writeSSE(c.Writer, SystemLogStreamEvent{
		Type:      "snapshot",
		Path:      logPath,
		Keyword:   keyword,
		Truncated: initial.Truncated,
		Total:     initial.Total,
		Lines:     initial.Lines,
	})
	flusher.Flush()

	ticker := time.NewTicker(streamPollInterval)
	defer ticker.Stop()

	var partial string
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			currentSize, sizeErr := getFileSize(logPath)
			if sizeErr != nil {
				if !errors.Is(sizeErr, os.ErrNotExist) {
					_ = writeSSE(c.Writer, SystemLogStreamEvent{
						Type:    "error",
						Message: sizeErr.Error(),
					})
					flusher.Flush()
				}
				continue
			}

			if currentSize < offset {
				offset = 0
				partial = ""
				lineNo = 0
				_ = writeSSE(c.Writer, SystemLogStreamEvent{
					Type:    "reset",
					Message: "日志文件已被截断，已从头重新读取",
				})
				flusher.Flush()
			}

			if currentSize == offset {
				continue
			}

			readSize := currentSize - offset
			if readSize > maxStreamReadBytes {
				readSize = maxStreamReadBytes
			}

			chunk, readErr := readFileChunk(logPath, offset, readSize)
			if readErr != nil {
				_ = writeSSE(c.Writer, SystemLogStreamEvent{
					Type:    "error",
					Message: readErr.Error(),
				})
				flusher.Flush()
				continue
			}
			offset += int64(len(chunk))

			batch := make([]SystemLogLine, 0, 64)
			content := partial + chunk
			parts := strings.Split(content, "\n")
			partial = parts[len(parts)-1]

			for i := 0; i < len(parts)-1; i++ {
				lineNo++
				lineText := parts[i]
				if keyword != "" && !strings.Contains(lineText, keyword) {
					continue
				}
				batch = append(batch, SystemLogLine{
					Line: lineNo,
					Text: lineText,
				})
			}

			if len(batch) > 0 {
				_ = writeSSE(c.Writer, SystemLogStreamEvent{
					Type:  "append",
					Lines: batch,
				})
				flusher.Flush()
			}
		}
	}
}

// Clear 清空日志文件内容（保留文件）。
func (h *SystemLogHandler) Clear(c *gin.Context) {
	logPath := infraConfig.GetServerLogPath()
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: fmt.Sprintf("创建日志目录失败: %v", err)})
		return
	}

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: fmt.Sprintf("打开日志文件失败: %v", err)})
		return
	}
	defer file.Close()

	if err := file.Truncate(0); err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: fmt.Sprintf("清空日志文件失败: %v", err)})
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		c.JSON(http.StatusInternalServerError, HTTPError{Code: http.StatusInternalServerError, Message: fmt.Sprintf("重置日志文件指针失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "日志文件已清空",
		"path":    logPath,
	})
}

// parseTailLines 解析并限制行数参数。
func parseTailLines(raw string) int {
	if strings.TrimSpace(raw) == "" {
		return defaultTailLines
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultTailLines
	}
	if n > maxTailLines {
		return maxTailLines
	}
	return n
}

type tailReadResult struct {
	Lines     []SystemLogLine
	Total     int64
	Truncated bool
}

// readTailLines 从日志中读取最后 N 行并支持关键字过滤。
func readTailLines(path string, lines int, keyword string) (*tailReadResult, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &tailReadResult{
				Lines:     []SystemLogLine{},
				Total:     0,
				Truncated: false,
			}, nil
		}
		return nil, fmt.Errorf("打开日志文件失败: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 2*1024*1024)

	result := make([]SystemLogLine, 0, lines)
	var total int64
	for scanner.Scan() {
		total++
		text := scanner.Text()
		if keyword != "" && !strings.Contains(text, keyword) {
			continue
		}
		result = append(result, SystemLogLine{
			Line: total,
			Text: text,
		})
		if len(result) > lines {
			result = result[1:]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取日志文件失败: %w", err)
	}

	return &tailReadResult{
		Lines:     result,
		Total:     total,
		Truncated: len(result) == lines,
	}, nil
}

// readFileChunk 按偏移读取日志文件片段。
func readFileChunk(path string, offset int64, readSize int64) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("打开日志文件失败: %w", err)
	}
	defer file.Close()

	buf := make([]byte, readSize)
	n, err := file.ReadAt(buf, offset)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("读取日志增量失败: %w", err)
	}
	return string(buf[:n]), nil
}

// getFileSize 获取文件大小。
func getFileSize(path string) (int64, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return stat.Size(), nil
}

// writeSSE 以 SSE 格式写入一条事件数据。
func writeSSE(w io.Writer, payload SystemLogStreamEvent) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "data: %s\n\n", data)
	return err
}
