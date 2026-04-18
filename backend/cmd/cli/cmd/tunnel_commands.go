package cmd

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/infrastructure/config"
)

const (
	tunnelPIDFileName = "tunnel.pid"
	tunnelLogFileName = "tunnel.log"
)

// saveTunnelConfig 保存 tunnel URL 到配置文件（使用 config 包的统一方法）
func saveTunnelConfig(publicURL string) error {
	return config.UpdatePublicURL(publicURL)
}

// getStoredPublicURL 从配置文件读取 public URL
func getStoredPublicURL() string {
	return config.GetPublicURL()
}

var tunnelURLRegex = regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "创建临时 Cloudflare Tunnel",
	Long:  "创建临时公共 URL，通过 Cloudflare Tunnel 访问本地服务器。\n\t无需配置 Cloudflare 账号，适合开发测试使用。\n\n\t需要先安装 cloudflared: https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/",
	Example: `  taskmanager tunnel start
  taskmanager tunnel start --port 8888
  taskmanager tunnel stop
  taskmanager tunnel status`,
}

var tunnelStartCmd = &cobra.Command{
	Use:   "start",
	Short: "在后台启动 Cloudflare Tunnel",
	Example: `  taskmanager tunnel start
  taskmanager tunnel start --port 8888`,
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		if port == 0 {
			port = config.GetServerPort()
		}

		// 检查 cloudflared 是否安装
		if !isCloudflaredInstalled() {
			fmt.Println("错误: 未安装 cloudflared")
			fmt.Println("请先安装: https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/")
			fmt.Println("macOS: brew install cloudflared")
			return
		}

		// 检查是否已在运行
		if isTunnelRunning() {
			fmt.Println("Tunnel 已在运行中")
			printTunnelStatus()
			return
		}

		// 检查服务器是否运行
		if !isRunning() {
			fmt.Println("错误: 服务器未运行，请先启动服务器: taskmanager server start")
			return
		}

		fmt.Printf("正在连接到 Cloudflare Tunnel...\n")
		fmt.Printf("本地服务器: http://localhost:%d\n", port)
		fmt.Println()

		// 清理残留进程并准备日志文件
		cleanupPreviousTunnel()
		time.Sleep(500 * time.Millisecond)

		configDir := getConfigDir()
		if err := os.MkdirAll(configDir, 0755); err != nil {
			fmt.Printf("创建配置目录失败: %v\n", err)
			return
		}

		logFile := getTunnelLogFile()
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			fmt.Printf("打开日志文件失败: %v\n", err)
			return
		}
		defer f.Close()

		// 后台启动 cloudflared
		command := exec.Command("cloudflared", "tunnel", "--url", fmt.Sprintf("http://localhost:%d", port))
		command.Stdout = f
		command.Stderr = f
		command.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
		}

		if err := command.Start(); err != nil {
			fmt.Printf("启动 cloudflared 失败: %v\n", err)
			return
		}

		pidFile := getTunnelPIDFilePath()
		if err := os.WriteFile(pidFile, []byte(strconv.Itoa(command.Process.Pid)), 0644); err != nil {
			fmt.Printf("写入 PID 文件失败: %v\n", err)
			_ = command.Process.Kill()
			return
		}

		// 轮询日志提取 URL（最多 30 秒）
		var tunnelURL string
		logFileReader, err := os.Open(logFile)
		if err != nil {
			fmt.Printf("打开日志文件读取失败: %v\n", err)
			return
		}
		defer logFileReader.Close()

		scanner := bufio.NewScanner(logFileReader)
		timeout := time.After(30 * time.Second)
		pollTick := time.NewTicker(500 * time.Millisecond)
		defer pollTick.Stop()

	pollLoop:
		for {
			select {
			case <-timeout:
				break pollLoop
			case <-pollTick.C:
				for scanner.Scan() {
					line := scanner.Text()
					if url := extractTunnelURL(line); url != "" {
						tunnelURL = url
						break pollLoop
					}
				}
				if err := scanner.Err(); err != nil {
					fmt.Printf("读取日志失败: %v\n", err)
					break pollLoop
				}
				// 重新打开文件继续读取新内容
				logFileReader.Close()
				var openErr error
				logFileReader, openErr = os.Open(logFile)
				if openErr != nil {
					fmt.Printf("重新打开日志文件失败: %v\n", openErr)
					continue
				}
				// 跳到末尾已读过的位置不太方便，简单方案：
				// 重新扫描全部（日志很短）
				scanner = bufio.NewScanner(logFileReader)
			}
		}

		if tunnelURL != "" {
			// 保存 tunnel URL 到配置文件
			if err := saveTunnelConfig(tunnelURL); err != nil {
				fmt.Printf("Warning: 保存 tunnel 配置失败: %v\n", err)
			}
			fmt.Println("=" + strings.Repeat("=", 50))
			fmt.Println("Tunnel 已创建成功!")
			fmt.Printf("公共 URL: %s\n", tunnelURL)
			fmt.Println()

			// 通知服务器更新所有 webhook URL
			if err := notifyServerToUpdateWebhooks(); err != nil {
				fmt.Printf("Warning: 通知服务器更新 webhook 失败: %v\n", err)
			} else {
				fmt.Println("已通知服务器更新 Webhook 地址")
			}

			fmt.Println()
			fmt.Println("使用 'taskmanager tunnel stop' 停止 Tunnel")
			fmt.Println("=" + strings.Repeat("=", 50))
			fmt.Println()
		} else {
			fmt.Println("Warning: 未能从日志中提取 Tunnel URL，请检查日志")
			fmt.Printf("日志文件: %s\n", logFile)
		}
	},
}

var tunnelStopCmd = &cobra.Command{
	Use:     "stop",
	Short:   "停止后台 Cloudflare Tunnel",
	Example: `  taskmanager tunnel stop`,
	Run: func(cmd *cobra.Command, args []string) {
		force, _ := cmd.Flags().GetBool("force")

		pid := getTunnelPID()
		if pid == 0 {
			fmt.Println("Tunnel 未运行")
			return
		}

		var err error
		if force {
			err = syscall.Kill(pid, syscall.SIGKILL)
		} else {
			err = syscall.Kill(pid, syscall.SIGTERM)
		}

		if err != nil {
			fmt.Printf("发送终止信号失败: %v\n", err)
			if strings.Contains(err.Error(), "no such process") {
				cleanupTunnelPIDFile()
				fmt.Println("Tunnel 已停止（PID 文件已清理）")
			}
			return
		}

		if !force {
			fmt.Println("正在停止 Tunnel...")
			for i := 0; i < 30; i++ {
				if !isProcessRunning(pid) {
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
		}

		if isProcessRunning(pid) && !force {
			fmt.Println("Tunnel 未能及时关闭，强制终止...")
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}

		cleanupTunnelPIDFile()
		fmt.Println("Tunnel 已停止")
	},
}

var tunnelStatusCmd = &cobra.Command{
	Use:     "status",
	Short:   "查看 Tunnel 运行状态",
	Example: `  taskmanager tunnel status`,
	Run: func(cmd *cobra.Command, args []string) {
		printTunnelStatus()
	},
}

// cleanupPreviousTunnel 清理上一次启动的 tunnel 进程
func cleanupPreviousTunnel() {
	pid := getTunnelPID()
	if pid == 0 {
		return
	}
	if isProcessRunning(pid) {
		_ = syscall.Kill(pid, syscall.SIGTERM)
		for i := 0; i < 20; i++ {
			if !isProcessRunning(pid) {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if isProcessRunning(pid) {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	}
	cleanupTunnelPIDFile()
}

// isCloudflaredInstalled 检查 cloudflared 是否已安装
func isCloudflaredInstalled() bool {
	_, err := exec.LookPath("cloudflared")
	return err == nil
}

// extractTunnelURL 从 cloudflared 输出中提取 Tunnel URL
func extractTunnelURL(line string) string {
	return tunnelURLRegex.FindString(line)
}

// getTunnelPIDFilePath 获取 Tunnel PID 文件路径
func getTunnelPIDFilePath() string {
	return filepath.Join(getConfigDir(), tunnelPIDFileName)
}

// getTunnelPID 获取 tunnel 进程 PID
func getTunnelPID() int {
	pidFile := getTunnelPIDFilePath()
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

// cleanupTunnelPIDFile 清理 Tunnel PID 文件
func cleanupTunnelPIDFile() {
	pidFile := getTunnelPIDFilePath()
	_ = os.Remove(pidFile)
}

// isTunnelRunning 检查 tunnel 是否正在运行
func isTunnelRunning() bool {
	pid := getTunnelPID()
	if pid == 0 {
		return false
	}
	return isProcessRunning(pid)
}

// getTunnelLogFile 获取 tunnel 日志文件路径
func getTunnelLogFile() string {
	return filepath.Join(getConfigDir(), tunnelLogFileName)
}

// printTunnelStatus 打印 tunnel 状态
func printTunnelStatus() {
	logFile := getTunnelLogFile()

	if !isTunnelRunning() {
		// 即使 tunnel 未运行，也尝试显示已存储的 URL
		if storedURL := getStoredPublicURL(); storedURL != "" {
			fmt.Println("Tunnel 状态: 未运行（但有已存储的 URL）")
			fmt.Printf("公共 URL: %s\n", storedURL)
		} else {
			fmt.Println("Tunnel 状态: 未运行")
		}
		fmt.Printf("日志文件: %s\n", logFile)
		return
	}

	pid := getTunnelPID()
	fmt.Println("Tunnel 状态: 运行中")
	fmt.Printf("PID: %d\n", pid)

	// 优先使用已存储的 URL（更可靠）
	storedURL := getStoredPublicURL()
	if storedURL != "" {
		fmt.Printf("公共 URL: %s\n", storedURL)
	} else {
		// 回退：从日志中提取
		if url := extractTunnelURLFromLog(logFile); url != "" {
			fmt.Printf("公共 URL: %s\n", url)
		}
	}
	fmt.Printf("日志文件: %s\n", logFile)
}

// extractTunnelURLFromLog 从日志文件中提取最新的 Tunnel URL
func extractTunnelURLFromLog(logFile string) string {
	file, err := os.Open(logFile)
	if err != nil {
		return ""
	}
	defer file.Close()

	// 读取全部行，找最后一个匹配的 URL
	var lastURL string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if url := extractTunnelURL(line); url != "" {
			lastURL = url
		}
	}
	return lastURL
}

// notifyServerToUpdateWebhooks 通知服务器更新所有 webhook URL
func notifyServerToUpdateWebhooks() error {
	// 读取服务器端口
	port := getServerHTTPPort()
	if port == 0 {
		port = 13618
	}

	url := fmt.Sprintf("http://localhost:%d/api/v1/internal/webhooks/update-all", port)
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		return fmt.Errorf("failed to notify server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}
	return nil
}

// getServerHTTPPort 获取服务器的 HTTP 端口
func getServerHTTPPort() int {
	// 尝试从配置文件读取
	cfg, err := config.Load()
	if err != nil {
		return 0
	}
	return cfg.Server.Port
}

// registerTunnelCommands 注册 tunnel 子命令
func registerTunnelCommands() {
	tunnelStartCmd.Flags().IntP("port", "p", 0, "本地服务器端口 (默认使用 SERVER_PORT 或 13618)")
	tunnelStopCmd.Flags().Bool("force", false, "强制停止 Tunnel")

	tunnelCmd.AddCommand(tunnelStartCmd)
	tunnelCmd.AddCommand(tunnelStopCmd)
	tunnelCmd.AddCommand(tunnelStatusCmd)
}
