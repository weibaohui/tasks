package cmd

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/domain"
	"github.com/weibh/taskmanager/infrastructure/config"
	_persistence "github.com/weibh/taskmanager/infrastructure/persistence"
)

// createAdminCmd 创建默认管理员用户
var createAdminCmd = &cobra.Command{
	Use:   "create-admin",
	Short: "创建默认管理员用户",
	Long: `创建默认管理员用户 (username: admin, password: admin123)

此命令直接操作数据库，无需启动 server。
首次部署后运行此命令创建管理员账号，然后使用 Web UI 登录。

注意：Token 需要在 Web UI 设置页面手动生成。`,
	Run: func(cmd *cobra.Command, args []string) {
		dbPath := config.GetDatabasePath()
		fmt.Printf("数据库路径: %s\n", dbPath)

		// 打开数据库连接
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			fmt.Printf("打开数据库失败: %v\n", err)
			return
		}
		defer db.Close()

		// 初始化数据库 Schema
		if err := _persistence.InitSchema(db); err != nil {
			fmt.Printf("初始化数据库结构失败: %v\n", err)
			return
		}

		// 检查是否已存在 admin 用户
		var exists int
		err = db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", "admin").Scan(&exists)
		if err != nil {
			fmt.Printf("检查用户失败: %v\n", err)
			return
		}
		if exists > 0 {
			fmt.Println("管理员用户已存在，无需重复创建。")
			fmt.Println("请使用 Web UI 登录。")
			return
		}

		// 生成 ID 和 Code
		id := generateID()
		code := generateCode()
		// 使用与 domain 一致的密码哈希方式
		passwordHash := domain.BuildStoredPasswordValue("admin123", "")

		// 插入用户记录
		now := time.Now().Unix()
		_, err = db.Exec(`
			INSERT INTO users (id, user_code, username, email, display_name, password_hash, is_active, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, code, "admin", "admin@local.dev", "系统管理员", passwordHash, 1, now, now)
		if err != nil {
			fmt.Printf("创建用户失败: %v\n", err)
			return
		}

		fmt.Println("管理员用户创建成功!")
		fmt.Println("")
		fmt.Println("登录信息:")
		fmt.Println("  用户名: admin")
		fmt.Println("  密码:   admin123 (请登录后立即修改)")
		fmt.Println("")
		fmt.Println("请使用 Web UI 登录，Token 需要在设置页面手动生成。")
	},
}

func registerCreateAdminCommands() {
}

// ID 生成器
var (
	counter uint64
	mu      sync.Mutex
)

const idChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// generateID 生成 21 字符的纳米 ID
func generateID() string {
	mu.Lock()
	counter++
	now := time.Now().UnixNano()
	counterVal := counter
	mu.Unlock()

	result := make([]byte, 21)
	num := now
	for i := 20; i >= 0; i-- {
		result[i] = idChars[num%62]
		num /= 62
	}
	// 添加 counter 混淆
	for i := 0; i < 21; i++ {
		result[i] = idChars[(int(result[i])+i+int(counterVal))%62]
	}
	return string(result)
}

// generateCode 生成用户 Code
func generateCode() string {
	mu.Lock()
	counter++
	now := time.Now().UnixNano()
	counterVal := counter
	mu.Unlock()

	result := make([]byte, 21)
	num := now
	for i := 20; i >= 0; i-- {
		result[i] = idChars[num%62]
		num /= 62
	}
	for i := 0; i < 21; i++ {
		result[i] = idChars[(int(result[i])+i+int(counterVal))%62]
	}
	return string(result)
}