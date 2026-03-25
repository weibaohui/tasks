/**
 * CLI 工具 - 用户管理
 */
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
	_persistence "github.com/weibh/taskmanager/infrastructure/persistence"
	"github.com/weibh/taskmanager/infrastructure/utils"
	"go.uber.org/zap"
)

const (
	DefaultAdminUsername = "admin"
	DefaultAdminPassword = "admin123"
)

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	createCmd := flag.NewFlagSet("create-admin", flag.ExitOnError)
	deleteCmd := flag.NewFlagSet("delete-admin", flag.ExitOnError)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "create-admin":
		createAdminUser(createCmd, logger)
	case "delete-admin":
		deleteAdminUser(deleteCmd, logger)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: cli <command> [options]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  create-admin    创建默认管理员用户 (admin/admin123)")
	fmt.Println("  delete-admin   删除默认管理员用户")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  cli create-admin")
	fmt.Println("  cli delete-admin")
}

func getDBAndRepos(logger *zap.Logger) (domain.UserRepository, domain.IDGenerator, func()) {
	dbPath := resolveDBPath()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		logger.Fatal("Failed to open database", zap.String("path", dbPath), zap.Error(err))
	}

	if err := _persistence.InitSchema(db); err != nil {
		logger.Fatal("Failed to init schema", zap.String("path", dbPath), zap.Error(err))
	}

	idGenerator := utils.NewNanoIDGenerator(21)
	userRepo := _persistence.NewSQLiteUserRepository(db)

	cleanup := func() {
		db.Close()
	}

	return userRepo, idGenerator, cleanup
}

func createAdminUser(cmd *flag.FlagSet, logger *zap.Logger) {
	userRepo, idGen, cleanup := getDBAndRepos(logger)
	defer cleanup()

	ctx := context.Background()

	// 检查是否已存在
	existingUser, err := userRepo.FindByUsername(ctx, DefaultAdminUsername)
	if err != nil {
		logger.Fatal("检查用户失败", zap.Error(err))
	}
	if existingUser != nil {
		logger.Info("管理员用户已存在", zap.String("username", DefaultAdminUsername))
		return
	}

	// 使用 application 层创建用户（会正确处理密码哈希）
	userService := application.NewUserApplicationService(userRepo, idGen)
	user, err := userService.CreateUser(ctx, application.CreateUserCommand{
		Username:    DefaultAdminUsername,
		DisplayName: "系统管理员",
		Email:       "admin@local.dev",
		Password:    DefaultAdminPassword,
	})
	if err != nil {
		logger.Fatal("创建管理员用户失败", zap.Error(err))
	}

	logger.Info("管理员用户创建成功",
		zap.String("username", user.Username()),
		zap.String("userCode", user.UserCode().String()),
	)
	fmt.Printf("初始密码: %s (请登录后立即修改)\n", DefaultAdminPassword)
}

func deleteAdminUser(cmd *flag.FlagSet, logger *zap.Logger) {
	userRepo, _, cleanup := getDBAndRepos(logger)
	defer cleanup()

	ctx := context.Background()

	// 查找用户
	existingUser, err := userRepo.FindByUsername(ctx, DefaultAdminUsername)
	if err != nil {
		logger.Fatal("查找用户失败", zap.Error(err))
	}
	if existingUser == nil {
		logger.Info("管理员用户不存在", zap.String("username", DefaultAdminUsername))
		return
	}

	// 删除用户
	if err := userRepo.Delete(ctx, existingUser.ID()); err != nil {
		logger.Fatal("删除管理员用户失败", zap.Error(err))
	}

	logger.Info("管理员用户已删除", zap.String("username", DefaultAdminUsername))
}

func resolveDBPath() string {
	if p := os.Getenv("TASKMANAGER_DB_PATH"); p != "" {
		return p
	}
	if p := os.Getenv("DB_PATH"); p != "" {
		return p
	}
	if st, err := os.Stat("./backend"); err == nil && st.IsDir() {
		return filepath.FromSlash("./backend/tasks.db")
	}
	return filepath.FromSlash("./tasks.db")
}
