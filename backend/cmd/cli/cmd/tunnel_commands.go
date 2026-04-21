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
	tunnelPIDFileName    = "tunnel.pid"
	tunnelLogFileName    = "tunnel.log"
	tunnelTypeFileName   = "tunnel.type"
	tunnelTypeHostc      = "hostc"
	tunnelTypeCloudflared = "cloudflared"
)

// tunnelTypeHostc and tunnelTypeCloudflared are constants defined above

// saveTunnelConfig 保存 tunnel URL 到配置文件（使用 config 包的统一方法）
func saveTunnelConfig(publicURL string) error {
	return config.UpdatePublicURL(publicURL)
}

// getStoredPublicURL 从配置文件读取 public URL
func getStoredPublicURL() string {
	return config.GetPublicURL()
}

var cloudflareTunnelURLRegex = regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "创建临时隧道访问本地服务器",
	Long: `创建临时公共 URL，通过隧道访问本地服务器。

默认使用 hostc 方式（如需使用 Cloudflare Tunnel，使用 'taskmanager tunnel cloudflared' 子命令）。

需要先安装 hostc: https://hostc.xyz`,
	Example: `  taskmanager tunnel start        # 默认使用 hostc
  taskmanager tunnel start --port 8888
  taskmanager tunnel stop
  taskmanager tunnel status
  taskmanager tunnel cloudflared start  # 使用 Cloudflare Tunnel`,
}

var tunnelStartCmd = &cobra.Command{
	Use:   "start",
	Short: "启动隧道（默认使用 hostc）",
	Example: `  taskmanager tunnel start
  taskmanager tunnel start --port 8888`,
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		if port == 0 {
			port = config.GetServerPort()
		}

		startHostcTunnel(port)
	},
}

var tunnelStopCmd = &cobra.Command{
	Use:     "stop",
	Short:   "停止后台隧道",
	Example: `  taskmanager tunnel stop`,
	Run: func(cmd *cobra.Command, args []string) {
		force, _ := cmd.Flags().GetBool("force")

		tunnelType := getTunnelType()
		if tunnelType == tunnelTypeCloudflared {
			stopCloudflaredTunnel(force)
		} else {
			stopHostcTunnel(force)
		}
	},
}

var tunnelStatusCmd = &cobra.Command{
	Use:     "status",
	Short:   "查看隧道运行状态",
	Example: `  taskmanager tunnel status`,
	Run: func(cmd *cobra.Command, args []string) {
		printTunnelStatus()
	},
}

// Cloudflare Tunnel 子命令
var tunnelCloudflaredCmd = &cobra.Command{
	Use:   "cloudflared",
	Short: "使用 Cloudflare Tunnel（传统方式）",
	Long:  "使用 Cloudflare Tunnel 创建公共 URL。\n需要安装 cloudflared: https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/",
	Example: `  taskmanager tunnel cloudflared start
  taskmanager tunnel cloudflared start --port 8888
  taskmanager tunnel cloudflared stop`,
}

var tunnelCloudflaredStartCmd = &cobra.Command{
	Use:   "start",
	Short: "启动 Cloudflare Tunnel",
	Example: `  taskmanager tunnel cloudflared start
  taskmanager tunnel cloudflared start --port 8888`,
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		if port == 0 {
			port = config.GetServerPort()
		}
		startCloudflaredTunnel(port)
	},
}

var tunnelCloudflaredStopCmd = &cobra.Command{
	Use:     "stop",
	Short:   "停止 Cloudflare Tunnel",
	Example: `  taskmanager tunnel cloudflared stop`,
	Run: func(cmd *cobra.Command, args []string) {
		force, _ := cmd.Flags().GetBool("force")
		stopCloudflaredTunnel(force)
	},
}

// ============================================================
// hostc tunnel 实现
// ============================================================

func startHostcTunnel(port int) {
	if !isHostcInstalled() {
		fmt.Println("错误: 未安装 hostc")
		fmt.Println("请先安装: https://hostc.xyz")
		return
	}

	if isTunnelRunning() {
		fmt.Println("Tunnel 已在运行中")
		printTunnelStatus()
		return
	}

	if !isRunning() {
		fmt.Println("错误: 服务器未运行，请先启动服务器: taskmanager server start")
		return
	}

	fmt.Printf("正在启动 hostc 隧道...\n")
	fmt.Printf("本地服务器: http://localhost:%d\n", port)
	fmt.Println()

	cleanupPreviousTunnel()
	time.Sleep(500 * time.Millisecond)

	configDir := getConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("创建配置目录失败: %v\n", err)
		return
	}

	logFile := getHostcLogFile()
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Printf("打开日志文件失败: %v\n", err)
		return
	}
	defer f.Close()

	// 后台启动 hostc
	command := exec.Command("hostc", strconv.Itoa(port))
	command.Stdout = f
	command.Stderr = f
	command.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	if err := command.Start(); err != nil {
		fmt.Printf("启动 hostc 失败: %v\n", err)
		return
	}

	// 保存 tunnel type
	setTunnelType(tunnelTypeHostc)

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
				if url := extractHostcURL(line); url != "" {
					tunnelURL = url
					break pollLoop
				}
			}
			if err := scanner.Err(); err != nil {
				fmt.Printf("读取日志失败: %v\n", err)
				break pollLoop
			}
			logFileReader.Close()
			var openErr error
			logFileReader, openErr = os.Open(logFile)
			if openErr != nil {
				fmt.Printf("重新打开日志文件失败: %v\n", openErr)
				continue
			}
			scanner = bufio.NewScanner(logFileReader)
		}
	}

	if tunnelURL != "" {
		if err := saveTunnelConfig(tunnelURL); err != nil {
			fmt.Printf("Warning: 保存 tunnel 配置失败: %v\n", err)
		}
		fmt.Println("=" + strings.Repeat("=", 50))
		fmt.Println("Tunnel 已创建成功!")
		fmt.Printf("公共 URL: %s\n", tunnelURL)
		fmt.Println()

		if err := notifyServerToUpdateWebhooks(); err != nil {
			fmt.Printf("Warning: 通知服务器更新 webhook 失败: %v\n", err)
		} else {
			fmt.Println("已通知服务器更新 Webhook 地址")
		}

		fmt.Println()
		fmt.Printf("使用 'taskmanager tunnel stop' 停止 Tunnel\n")
		fmt.Println("=" + strings.Repeat("=", 50))
		fmt.Println()
	} else {
		fmt.Println("Warning: 未能从日志中提取 Tunnel URL，请检查日志")
		fmt.Printf("日志文件: %s\n", logFile)
	}
}

func stopHostcTunnel(force bool) {
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
			cleanupTunnelTypeFile()
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
	cleanupTunnelTypeFile()
	fmt.Println("Tunnel 已停止")
}

func extractHostcURL(line string) string {
	// hostc 可能输出的 URL 格式类似: https://xxx.hostc.io 或直接显示 URL
	if strings.Contains(line, "https://") && strings.Contains(line, ".trycloudflare.com") {
		return cloudflareTunnelURLRegex.FindString(line)
	}
	// hostc 可能有其他输出格式，尝试通用匹配
	if strings.Contains(line, "->") {
		parts := strings.Split(line, "->")
		if len(parts) >= 2 {
			url := strings.TrimSpace(parts[len(parts)-1])
			if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
				return strings.Fields(url)[0]
			}
		}
	}
	return ""
}

func getHostcLogFile() string {
	return filepath.Join(getConfigDir(), "hostc.log")
}

func isHostcInstalled() bool {
	_, err := exec.LookPath("hostc")
	return err == nil
}

// ============================================================
// Cloudflare Tunnel 实现
// ============================================================

func startCloudflaredTunnel(port int) {
	if !isCloudflaredInstalled() {
		fmt.Println("错误: 未安装 cloudflared")
		fmt.Println("请先安装: https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/")
		fmt.Println("macOS: brew install cloudflared")
		return
	}

	if isTunnelRunning() {
		fmt.Println("Tunnel 已在运行中")
		printTunnelStatus()
		return
	}

	if !isRunning() {
		fmt.Println("错误: 服务器未运行，请先启动服务器: taskmanager server start")
		return
	}

	fmt.Printf("正在连接到 Cloudflare Tunnel...\n")
	fmt.Printf("本地服务器: http://localhost:%d\n", port)
	fmt.Println()

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

	// 保存 tunnel type
	setTunnelType(tunnelTypeCloudflared)

	pidFile := getTunnelPIDFilePath()
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(command.Process.Pid)), 0644); err != nil {
		fmt.Printf("写入 PID 文件失败: %v\n", err)
		_ = command.Process.Kill()
		return
	}

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
				if url := extractCloudflareTunnelURL(line); url != "" {
					tunnelURL = url
					break pollLoop
				}
			}
			if err := scanner.Err(); err != nil {
				fmt.Printf("读取日志失败: %v\n", err)
				break pollLoop
			}
			logFileReader.Close()
			var openErr error
			logFileReader, openErr = os.Open(logFile)
			if openErr != nil {
				fmt.Printf("重新打开日志文件失败: %v\n", openErr)
				continue
			}
			scanner = bufio.NewScanner(logFileReader)
		}
	}

	if tunnelURL != "" {
		if err := saveTunnelConfig(tunnelURL); err != nil {
			fmt.Printf("Warning: 保存 tunnel 配置失败: %v\n", err)
		}
		fmt.Println("=" + strings.Repeat("=", 50))
		fmt.Println("Tunnel 已创建成功!")
		fmt.Printf("公共 URL: %s\n", tunnelURL)
		fmt.Println()

		if err := notifyServerToUpdateWebhooks(); err != nil {
			fmt.Printf("Warning: 通知服务器更新 webhook 失败: %v\n", err)
		} else {
			fmt.Println("已通知服务器更新 Webhook 地址")
		}

		fmt.Println()
		fmt.Printf("使用 'taskmanager tunnel cloudflared stop' 停止 Tunnel\n")
		fmt.Println("=" + strings.Repeat("=", 50))
		fmt.Println()
	} else {
		fmt.Println("Warning: 未能从日志中提取 Tunnel URL，请检查日志")
		fmt.Printf("日志文件: %s\n", logFile)
	}
}

func stopCloudflaredTunnel(force bool) {
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
			cleanupTunnelTypeFile()
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
	cleanupTunnelTypeFile()
	fmt.Println("Tunnel 已停止")
}

func extractCloudflareTunnelURL(line string) string {
	return cloudflareTunnelURLRegex.FindString(line)
}

// ============================================================
// 共享函数
// ============================================================

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

// getTunnelTypeFilePath 获取 tunnel type 文件路径
func getTunnelTypeFilePath() string {
	return filepath.Join(getConfigDir(), tunnelTypeFileName)
}

// getTunnelType 获取当前 tunnel 类型
func getTunnelType() string {
	typeFile := getTunnelTypeFilePath()
	data, err := os.ReadFile(typeFile)
	if err != nil {
		return tunnelTypeHostc // 默认为 hostc
	}
	return strings.TrimSpace(string(data))
}

// setTunnelType 设置当前 tunnel 类型
func setTunnelType(t string) {
	typeFile := getTunnelTypeFilePath()
	_ = os.WriteFile(typeFile, []byte(t), 0644)
}

// cleanupTunnelTypeFile 清理 Tunnel Type 文件
func cleanupTunnelTypeFile() {
	typeFile := getTunnelTypeFilePath()
	_ = os.Remove(typeFile)
}

// printTunnelStatus 打印 tunnel 状态
func printTunnelStatus() {
	logFile := getTunnelLogFile()
	hostcLogFile := getHostcLogFile()
	tunnelType := getTunnelType()

	if tunnelType == tunnelTypeCloudflared {
		logFile = getTunnelLogFile()
	} else {
		logFile = hostcLogFile
	}

	if !isTunnelRunning() {
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
	fmt.Printf("Tunnel 状态: 运行中 (类型: %s)\n", tunnelType)
	fmt.Printf("PID: %d\n", pid)

	storedURL := getStoredPublicURL()
	if storedURL != "" {
		fmt.Printf("公共 URL: %s\n", storedURL)
	} else {
		var url string
		if tunnelType == tunnelTypeCloudflared {
			url = extractCloudflareTunnelURLFromLog(logFile)
		} else {
			url = extractHostcURLFromLog(logFile)
		}
		if url != "" {
			fmt.Printf("公共 URL: %s\n", url)
		}
	}
	fmt.Printf("日志文件: %s\n", logFile)
}

// extractTunnelURLFromLog 从日志文件中提取最新的 Tunnel URL (cloudflare)
func extractCloudflareTunnelURLFromLog(logFile string) string {
	file, err := os.Open(logFile)
	if err != nil {
		return ""
	}
	defer file.Close()

	var lastURL string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if url := extractCloudflareTunnelURL(line); url != "" {
			lastURL = url
		}
	}
	return lastURL
}

// extractHostcURLFromLog 从日志文件中提取最新的 Tunnel URL (hostc)
func extractHostcURLFromLog(logFile string) string {
	file, err := os.Open(logFile)
	if err != nil {
		return ""
	}
	defer file.Close()

	var lastURL string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if url := extractHostcURL(line); url != "" {
			lastURL = url
		}
	}
	return lastURL
}

// notifyServerToUpdateWebhooks 通知服务器更新所有 webhook URL
func notifyServerToUpdateWebhooks() error {
	port := getServerHTTPPort()
	if port == 0 {
		port = 13618
	}

	token := config.GetAPIToken()
	if token == "" {
		return fmt.Errorf("API token is empty, cannot authenticate")
	}

	url := fmt.Sprintf("http://localhost:%d/api/v1/internal/webhooks/update-all", port)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
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

	tunnelCloudflaredStartCmd.Flags().IntP("port", "p", 0, "本地服务器端口 (默认使用 SERVER_PORT 或 13618)")
	tunnelCloudflaredStopCmd.Flags().Bool("force", false, "强制停止 Tunnel")

	tunnelCmd.AddCommand(tunnelStartCmd)
	tunnelCmd.AddCommand(tunnelStopCmd)
	tunnelCmd.AddCommand(tunnelStatusCmd)

	tunnelCloudflaredCmd.AddCommand(tunnelCloudflaredStartCmd)
	tunnelCloudflaredCmd.AddCommand(tunnelCloudflaredStopCmd)
	tunnelCmd.AddCommand(tunnelCloudflaredCmd)
}
