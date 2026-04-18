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
	infraConfig "github.com/weibh/taskmanager/infrastructure/config"
)

const (
	pidFileName = "server.pid"
	logFileName = "server.log"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "服务器管理",
	Long:  `在后台启动、停止、查看 TaskManager 服务器状态`,
}

var serverStartCmd = &cobra.Command{
	Use:   "start",
	Short: "在后台启动服务器",
	Example: `  taskmanager server start
  taskmanager server start --port 8888`,
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")

		// 检查是否已在运行
		if isRunning() {
			fmt.Println("服务器已在运行中")
			printStatus()
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

		// 找到真正的 server 可执行文件
		serverExe := findServerExecutable(exePath)
		if serverExe == "" {
			fmt.Println("未找到 server 可执行文件，请确保已编译: make build")
			return
		}

		// 设置环境变量
		env := os.Environ()
		if port != 0 {
			env = append(env, fmt.Sprintf("SERVER_PORT=%d", port))
		}
		// 传递 TASKMANAGER_DB_PATH（如果存在）
		if dbPath := infraConfig.GetEnv("TASKMANAGER_DB_PATH"); dbPath != "" {
			env = append(env, fmt.Sprintf("TASKMANAGER_DB_PATH=%s", dbPath))
		}

		// 打开日志文件
		logFile := getServerLogPath()
		logDir := filepath.Dir(logFile)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			fmt.Printf("创建日志目录失败: %v\n", err)
			return
		}
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Printf("打开日志文件失败: %v\n", err)
			return
		}
		defer f.Close()

		// 启动后台进程
		command := exec.Command(serverExe)
		command.Env = env
		command.Stdout = f
		command.Stderr = f
		command.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true, // 创建新的进程组，防止被子进程继承信号
		}

		if err := command.Start(); err != nil {
			fmt.Printf("启动服务器失败: %v\n", err)
			return
		}

		// 写入 PID 文件
		pidFile := filepath.Join(configDir, pidFileName)
		if err := os.WriteFile(pidFile, []byte(strconv.Itoa(command.Process.Pid)), 0644); err != nil {
			fmt.Printf("写入 PID 文件失败: %v\n", err)
			// 尝试终止已启动的进程
			_ = command.Process.Kill()
			return
		}

		// 等待一小段时间检查是否成功启动
		time.Sleep(500 * time.Millisecond)

		if isRunning() {
			fmt.Println("服务器启动成功")
			printStatus()
		} else {
			fmt.Println("服务器可能启动失败，请检查日志")
			fmt.Printf("日志文件: %s\n", logFile)
		}
	},
}

var serverStopCmd = &cobra.Command{
	Use:     "stop",
	Short:   "停止后台服务器",
	Example: `  taskmanager server stop`,
	Run: func(cmd *cobra.Command, args []string) {
		force, _ := cmd.Flags().GetBool("force")

		pid := getPID()
		if pid == 0 {
			fmt.Println("服务器未运行")
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
				cleanupPIDFile()
				fmt.Println("服务器已停止（PID 文件已清理）")
			}
			return
		}

		// 等待进程结束
		if !force {
			fmt.Println("正在优雅关闭服务器...")
			for i := 0; i < 30; i++ {
				if !isProcessRunning(pid) {
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
		}

		// 如果还在运行，强制终止
		if isProcessRunning(pid) && !force {
			fmt.Println("服务器未能及时关闭，强制终止...")
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}

		cleanupPIDFile()
		fmt.Println("服务器已停止")
	},
}

var serverStatusCmd = &cobra.Command{
	Use:     "status",
	Short:   "查看服务器状态",
	Example: `  taskmanager server status`,
	Run: func(cmd *cobra.Command, args []string) {
		printStatus()
	},
}

var serverLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "查看服务器日志",
	Example: `  taskmanager server logs
  taskmanager server logs -f  # 实时跟踪日志
  taskmanager server logs -n 100  # 查看最后 100 行`,
	Run: func(cmd *cobra.Command, args []string) {
		follow, _ := cmd.Flags().GetBool("follow")
		lines, _ := cmd.Flags().GetInt("lines")

		logFile := getServerLogPath()

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

var serverRestartCmd = &cobra.Command{
	Use:     "restart",
	Short:   "重启服务器",
	Example: `  taskmanager server restart`,
	Run: func(cmd *cobra.Command, args []string) {
		// 1. 先尝试按 PID 文件停止
		pid := getPID()
		if pid != 0 {
			fmt.Println("正在停止服务器...")
			_ = syscall.Kill(pid, syscall.SIGTERM)
			for i := 0; i < 30; i++ {
				if !isProcessRunning(pid) {
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
			if isProcessRunning(pid) {
				_ = syscall.Kill(pid, syscall.SIGKILL)
			}
			cleanupPIDFile()
			fmt.Println("服务器已停止")
		}

		// 2. 检查端口是否仍被占用（处理 PID 文件过期的情况）
		serverPort := getServerPort()
		if pid, ok := findProcessOnPort(serverPort); ok {
			fmt.Printf("端口 %d 仍被进程 %d 占用，正在终止...\n", serverPort, pid)
			_ = syscall.Kill(pid, syscall.SIGTERM)
			for i := 0; i < 30; i++ {
				if !isProcessRunning(pid) {
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
			if isProcessRunning(pid) {
				_ = syscall.Kill(pid, syscall.SIGKILL)
				time.Sleep(200 * time.Millisecond)
			}
			fmt.Printf("进程 %d 已终止\n", pid)
		}

		// 再启动
		fmt.Println("正在启动服务器...")
		serverStartCmd.Run(cmd, args)
	},
}
