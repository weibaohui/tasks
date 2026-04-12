package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

var tunnelCmd = &cobra.Command{
	Use:   "tunnel",
	Short: "创建临时 Cloudflare Tunnel",
	Long: "创建临时公共 URL，通过 Cloudflare Tunnel 访问本地服务器。\n\t无需配置 Cloudflare 账号，适合开发测试使用。\n\n\t需要先安装 cloudflared: https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/",
	Example: `  taskmanager tunnel
	  taskmanager tunnel --port 8888`,
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		if port == 0 {
			port = getServerPort()
		}

		// 检查 cloudflared 是否安装
		if !isCloudflaredInstalled() {
			fmt.Println("错误: 未安装 cloudflared")
			fmt.Println("请先安装: https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/")
			fmt.Println("macOS: brew install cloudflared")
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

		// 启动 cloudflared tunnel
		url := runCloudflaredTunnel(port)
		if url == "" {
			fmt.Println("创建 Tunnel 失败")
			return
		}

		fmt.Println()
		fmt.Println("=" + strings.Repeat("=", 50))
		fmt.Println("Tunnel 已创建成功!")
		fmt.Printf("公共 URL: %s\n", url)
		fmt.Println()
		fmt.Println("按 Ctrl+C 停止 Tunnel")
		fmt.Println("=" + strings.Repeat("=", 50))
		fmt.Println()

		// 等待中断信号
		waitForInterrupt()
	},
}

// isCloudflaredInstalled 检查 cloudflared 是否已安装
func isCloudflaredInstalled() bool {
	_, err := exec.LookPath("cloudflared")
	return err == nil
}

// runCloudflaredTunnel 启动 cloudflared tunnel 并返回 URL
func runCloudflaredTunnel(port int) string {
	cmd := exec.Command("cloudflared", "tunnel", "--url", fmt.Sprintf("http://localhost:%d", port))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 创建管道用于捕获输出
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		fmt.Printf("启动 cloudflared 失败: %v\n", err)
		return ""
	}

	// 异步读取输出，提取 URL
	var tunnelURL string
	done := make(chan bool)

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line)
			if url := extractTunnelURL(line); url != "" {
				tunnelURL = url
			}
		}
		done <- true
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line)
			if url := extractTunnelURL(line); url != "" {
				tunnelURL = url
			}
		}
		done <- true
	}()

	// 等待命令完成或被中断
	waitDone := false
	for !waitDone {
		select {
		case <-done:
			waitDone = true
		default:
			// 检查进程是否退出
			if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
				return ""
			}
		}
	}

	return tunnelURL
}

// extractTunnelURL 从 cloudflared 输出中提取 Tunnel URL
func extractTunnelURL(line string) string {
	// cloudflared 输出格式: https://randomname.trycloudflare.com
	// 或者: .*https://[a-z0-9-]+\.trycloudflare\.com.*
	re := regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)
	matches := re.FindString(line)
	return matches
}

// waitForInterrupt 等待中断信号
func waitForInterrupt() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	fmt.Println("\n正在停止 Tunnel...")
}

// registerTunnelCommands 注册 tunnel 子命令
func registerTunnelCommands() {
	tunnelCmd.Flags().IntP("port", "p", 0, "本地服务器端口 (默认使用 SERVER_PORT 或 13618)")
}