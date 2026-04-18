package cmd

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	infraConfig "github.com/weibh/taskmanager/infrastructure/config"
)

// getConfigDir 获取配置目录
func getConfigDir() string {
	return infraConfig.GetConfigDir()
}

// getPIDFilePath 获取 PID 文件路径
func getPIDFilePath() string {
	return filepath.Join(getConfigDir(), pidFileName)
}

// getServerLogPath 获取服务日志文件路径
func getServerLogPath() string {
	if path := infraConfig.GetServerLogPathFromEnv(); path != "" {
		return path
	}
	return infraConfig.GetServerLogPath()
}

// getPID 获取服务器进程 PID
func getPID() int {
	pidFile := getPIDFilePath()
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0
	}
	return pid
}

// isProcessRunning 检查进程是否正在运行
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// 发送信号 0 检查进程是否存在
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// isRunning 检查服务器是否正在运行
func isRunning() bool {
	pid := getPID()
	if pid == 0 {
		return false
	}
	return isProcessRunning(pid)
}

// cleanupPIDFile 清理 PID 文件
func cleanupPIDFile() {
	pidFile := getPIDFilePath()
	_ = os.Remove(pidFile)
}

// getServerPort 获取服务器端口
func getServerPort() int {
	return infraConfig.GetServerPortFromEnv()
}

// findProcessOnPort 查找占用指定端口的进程 PID
func findProcessOnPort(port int) (int, bool) {
	// 尝试监听端口，如果成功说明端口空闲
	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err == nil {
		ln.Close()
		return 0, false
	}

	// 端口被占用，使用 lsof 查找占用进程
	cmd := exec.Command("lsof", "-ti", fmt.Sprintf(":%d", port), "-sTCP:LISTEN")
	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return 0, false
	}
	pidStr := strings.TrimSpace(string(output))
	// lsof 可能返回多行（多个 PID），取第一行
	lines := strings.Split(pidStr, "\n")
	pid, err := strconv.Atoi(strings.TrimSpace(lines[0]))
	if err != nil {
		return 0, false
	}
	return pid, true
}

// printStatus 打印服务器状态
func printStatus() {
	if !isRunning() {
		fmt.Println("服务器状态: 未运行")
		return
	}

	pid := getPID()
	fmt.Printf("服务器状态: 运行中\n")
	fmt.Printf("PID: %d\n", pid)

	// 尝试读取端口
	port := infraConfig.GetServerPortFromEnv()
	fmt.Printf("服务地址: http://localhost:%d\n", port)

	// 显示日志文件路径
	logFile := getServerLogPath()
	fmt.Printf("日志文件: %s\n", logFile)
}

// findServerExecutable 查找 server 可执行文件
func findServerExecutable(cliPath string) string {
	home, _ := os.UserHomeDir()

	// 可能的 server 可执行文件路径（按优先级）
	possiblePaths := []string{
		// 与 CLI 同目录下的 server
		filepath.Join(filepath.Dir(cliPath), "taskmanager-server"),
		// 用户本地安装路径
		filepath.Join(home, "bin", "taskmanager-server"),
		// 标准系统安装路径（兼容旧版本）
		"/usr/local/bin/taskmanager-server",
		// 开发环境路径
		"./backend/bin/taskmanager-server",
		"./bin/taskmanager-server",
		// 相对于 CLI 的上级目录
		filepath.Join(filepath.Dir(cliPath), "..", "bin", "taskmanager-server"),
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			// 检查是否是可执行文件
			info, err := os.Stat(path)
			if err != nil {
				continue
			}
			// Unix 可执行权限检查
			if info.Mode()&0111 != 0 {
				return path
			}
		}
	}

	// 尝试使用 go run 直接运行（开发环境）
	if _, err := os.Stat("./backend/cmd/server/main.go"); err == nil {
		return "go"
	}

	return ""
}
