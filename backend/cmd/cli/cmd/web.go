package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

const (
	webPIDFileName = "web.pid"
	webLogFileName = "web.log"
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Web 管理服务",
	Long:  `在后台启动、停止、查看 TaskManager Web 管理服务（HTTP API + 前端）`,
}

var webStartCmd = &cobra.Command{
	Use:   "start",
	Short: "在后台启动 Web 服务",
	Example: `  taskmanager web start
  taskmanager web start --port 8888`,
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")

		// 检查是否已在运行
		if isWebRunning() {
			fmt.Println("Web 服务已在运行中")
			printWebStatus()
			return
		}

		// 确保配置目录存在
		configDir := getConfigDir()
		if err := os.MkdirAll(configDir, 0755); err != nil {
			fmt.Printf("创建配置目录失败: %v\n", err)
			return
		}

		// 获取可执行文件路径
		exePath, err := os.Executable()
		if err != nil {
			fmt.Printf("获取可执行文件路径失败: %v\n", err)
			return
		}

		// 找到真正的 web 可执行文件
		webExe := findWebExecutable(exePath)
		if webExe == "" {
			fmt.Println("未找到 web 可执行文件，请确保已编译: make build")
			return
		}

		// 设置环境变量
		env := os.Environ()
		if port != 0 {
			env = append(env, fmt.Sprintf("SERVER_PORT=%d", port))
		}

		// 打开日志文件
		logFile := filepath.Join(configDir, webLogFileName)
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Printf("打开日志文件失败: %v\n", err)
			return
		}
		defer f.Close()

		// 启动后台进程
		command := exec.Command(webExe)
		command.Env = env
		command.Stdout = f
		command.Stderr = f
		command.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true, // 创建新的进程组，防止被子进程继承信号
		}

		if err := command.Start(); err != nil {
			fmt.Printf("启动 Web 服务失败: %v\n", err)
			return
		}

		// 写入 PID 文件
		pidFile := filepath.Join(configDir, webPIDFileName)
		if err := os.WriteFile(pidFile, []byte(strconv.Itoa(command.Process.Pid)), 0644); err != nil {
			fmt.Printf("写入 PID 文件失败: %v\n", err)
			// 尝试终止已启动的进程
			_ = command.Process.Kill()
			return
		}

		// 等待一小段时间检查是否成功启动
		time.Sleep(500 * time.Millisecond)

		if isWebRunning() {
			fmt.Println("Web 服务启动成功")
			printWebStatus()
		} else {
			fmt.Println("Web 服务可能启动失败，请检查日志")
			fmt.Printf("日志文件: %s\n", logFile)
		}
	},
}

var webStopCmd = &cobra.Command{
	Use:     "stop",
	Short:   "停止后台 Web 服务",
	Example: `  taskmanager web stop`,
	Run: func(cmd *cobra.Command, args []string) {
		force, _ := cmd.Flags().GetBool("force")

		pid := getWebPID()
		if pid == 0 {
			fmt.Println("Web 服务未运行")
			return
		}

		// 发送终止信号
		var err error
		if force {
			err = syscall.Kill(pid, syscall.SIGKILL)
		} else {
			err = syscall.Kill(pid, syscall.SIGTERM)
		}

		if err != nil {
			fmt.Printf("发送终止信号失败: %v\n", err)
			// 如果进程不存在，清理 PID 文件
			if strings.Contains(err.Error(), "no such process") {
				cleanupWebPIDFile()
				fmt.Println("Web 服务已停止（PID 文件已清理）")
			}
			return
		}

		// 等待进程结束
		if !force {
			fmt.Println("正在优雅关闭 Web 服务...")
			for i := 0; i < 30; i++ {
				if !isWebProcessRunning(pid) {
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
		}

		// 如果还在运行，强制终止
		if isWebProcessRunning(pid) && !force {
			fmt.Println("Web 服务未能及时关闭，强制终止...")
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}

		cleanupWebPIDFile()
		fmt.Println("Web 服务已停止")
	},
}

var webStatusCmd = &cobra.Command{
	Use:     "status",
	Short:   "查看 Web 服务状态",
	Example: `  taskmanager web status`,
	Run: func(cmd *cobra.Command, args []string) {
		printWebStatus()
	},
}

var webLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "查看 Web 服务日志",
	Example: `  taskmanager web logs
  taskmanager web logs -f  # 实时跟踪日志
  taskmanager web logs -n 100  # 查看最后 100 行`,
	Run: func(cmd *cobra.Command, args []string) {
		follow, _ := cmd.Flags().GetBool("follow")
		lines, _ := cmd.Flags().GetInt("lines")

		configDir := getConfigDir()
		logFile := filepath.Join(configDir, webLogFileName)

		// 检查日志文件是否存在
		if _, err := os.Stat(logFile); os.IsNotExist(err) {
			fmt.Println("日志文件不存在")
			return
		}

		if follow {
			// 使用 tail -f 实时跟踪
			tailCmd := exec.Command("tail", "-f", "-n", strconv.Itoa(lines), logFile)
			tailCmd.Stdout = os.Stdout
			tailCmd.Stderr = os.Stderr
			_ = tailCmd.Run()
		} else {
			// 读取指定行数
			file, err := os.Open(logFile)
			if err != nil {
				fmt.Printf("打开日志文件失败: %v\n", err)
				return
			}
			defer file.Close()

			// 简单实现：读取全部，输出最后 N 行
			content, err := io.ReadAll(file)
			if err != nil {
				fmt.Printf("读取日志文件失败: %v\n", err)
				return
			}

			allLines := strings.Split(string(content), "\n")
			start := 0
			if len(allLines) > lines {
				start = len(allLines) - lines
			}
			for i := start; i < len(allLines); i++ {
				fmt.Println(allLines[i])
			}
		}
	},
}

var webRestartCmd = &cobra.Command{
	Use:     "restart",
	Short:   "重启 Web 服务",
	Example: `  taskmanager web restart`,
	Run: func(cmd *cobra.Command, args []string) {
		// 先停止
		pid := getWebPID()
		if pid != 0 {
			fmt.Println("正在停止 Web 服务...")
			_ = syscall.Kill(pid, syscall.SIGTERM)
			for i := 0; i < 30; i++ {
				if !isWebProcessRunning(pid) {
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
			if isWebProcessRunning(pid) {
				_ = syscall.Kill(pid, syscall.SIGKILL)
			}
			cleanupWebPIDFile()
			fmt.Println("Web 服务已停止")
		}

		// 再启动
		fmt.Println("正在启动 Web 服务...")
		webStartCmd.Run(cmd, args)
	},
}

// getWebPIDFilePath 获取 Web PID 文件路径
func getWebPIDFilePath() string {
	return filepath.Join(getConfigDir(), webPIDFileName)
}

// getWebPID 获取 Web 进程 PID
func getWebPID() int {
	pidFile := getWebPIDFilePath()
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

// isWebProcessRunning 检查 Web 进程是否正在运行
func isWebProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// 发送信号 0 检查进程是否存在
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// isWebRunning 检查 Web 服务是否正在运行
func isWebRunning() bool {
	pid := getWebPID()
	if pid == 0 {
		return false
	}
	return isWebProcessRunning(pid)
}

// cleanupWebPIDFile 清理 Web PID 文件
func cleanupWebPIDFile() {
	pidFile := getWebPIDFilePath()
	_ = os.Remove(pidFile)
}

// printWebStatus 打印 Web 服务状态
func printWebStatus() {
	if !isWebRunning() {
		fmt.Println("Web 服务状态: 未运行")
		return
	}

	pid := getWebPID()
	fmt.Printf("Web 服务状态: 运行中\n")
	fmt.Printf("PID: %d\n", pid)

	// 尝试读取端口
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "13618" // 正式环境默认端口
	}
	fmt.Printf("Web 地址: http://localhost:%s\n", port)

	// 显示日志文件路径
	logFile := filepath.Join(getConfigDir(), webLogFileName)
	fmt.Printf("日志文件: %s\n", logFile)
}

// findWebExecutable 查找 web 可执行文件
func findWebExecutable(cliPath string) string {
	// 可能的 web 可执行文件路径（按优先级）
	possiblePaths := []string{
		// 与 CLI 同目录下的 web
		filepath.Join(filepath.Dir(cliPath), "taskmanager-web"),
		// 标准安装路径
		"/usr/local/bin/taskmanager-web",
		// 开发环境路径
		"./backend/bin/taskmanager-web",
		"./bin/taskmanager-web",
		// 相对于 CLI 的上级目录
		filepath.Join(filepath.Dir(cliPath), "..", "bin", "taskmanager-web"),
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
	if _, err := os.Stat("./backend/cmd/web/main.go"); err == nil {
		return "go"
	}

	return ""
}

func init() {
	webStartCmd.Flags().Int("port", 0, "Web 服务端口（默认 8888）")
	webStopCmd.Flags().Bool("force", false, "强制停止 Web 服务")
	webLogsCmd.Flags().BoolP("follow", "f", false, "实时跟踪日志输出")
	webLogsCmd.Flags().IntP("lines", "n", 50, "显示最后 N 行日志")

	webCmd.AddCommand(webStartCmd)
	webCmd.AddCommand(webStopCmd)
	webCmd.AddCommand(webStatusCmd)
	webCmd.AddCommand(webLogsCmd)
	webCmd.AddCommand(webRestartCmd)

	rootCmd.AddCommand(webCmd)
}
