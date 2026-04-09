package cmd

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
	"github.com/weibh/taskmanager/infrastructure/config"
	_persistence "github.com/weibh/taskmanager/infrastructure/persistence"
)

// deleteAdminCmd 删除默认管理员用户
var deleteAdminCmd = &cobra.Command{
	Use:   "delete-admin",
	Short: "删除默认管理员用户",
	Long: `删除默认管理员用户

此命令直接操作数据库，无需启动 server。
警告：删除后所有数据将被永久清除！`,
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

		// 开始事务
		tx, err := db.Begin()
		if err != nil {
			fmt.Printf("开启事务失败: %v\n", err)
			return
		}
		defer tx.Rollback()

		// 查询管理员 ID
		var adminID string
		if err := tx.QueryRow("SELECT id FROM users WHERE username = ?", "admin").Scan(&adminID); err != nil {
			if err == sql.ErrNoRows {
				fmt.Println("管理员用户不存在，无需删除。")
				return
			}
			fmt.Printf("查询管理员失败: %v\n", err)
			return
		}

		// 先删除 token
		if _, err = tx.Exec("DELETE FROM user_tokens WHERE user_id = ?", adminID); err != nil {
			fmt.Printf("删除 token 失败: %v\n", err)
			return
		}

		// 再删除用户
		if _, err = tx.Exec("DELETE FROM users WHERE id = ?", adminID); err != nil {
			fmt.Printf("删除用户失败: %v\n", err)
			return
		}

		// 提交事务
		if err := tx.Commit(); err != nil {
			fmt.Printf("提交事务失败: %v\n", err)
			return
		}

		fmt.Println("管理员用户已删除。")
	},
}

func init() {
	rootCmd.AddCommand(deleteAdminCmd)
}