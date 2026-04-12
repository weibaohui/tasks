package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/weibh/taskmanager/application"
	"github.com/weibh/taskmanager/domain"
	_persistence "github.com/weibh/taskmanager/infrastructure/persistence"
	"github.com/weibh/taskmanager/infrastructure/utils"
	"go.uber.org/zap"
)

func runAdminSubcommandIfMatched(logger *zap.Logger) bool {
	if len(os.Args) < 2 {
		return false
	}

	switch os.Args[1] {
	case "create-admin":
		if err := runCreateAdmin(logger); err != nil {
			logger.Fatal("创建默认管理员用户失败", zap.Error(err))
		}
		return true
	case "delete-admin":
		if err := runDeleteAdmin(logger); err != nil {
			logger.Fatal("删除默认管理员用户失败", zap.Error(err))
		}
		return true
	default:
		return false
	}
}

func runCreateAdmin(logger *zap.Logger) error {
	userRepo, idGen, cleanup, err := getDBAndUserRepo(logger)
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := context.Background()
	existingUser, err := userRepo.FindByUsername(ctx, DefaultAdminUsername)
	if err != nil {
		return fmt.Errorf("检查用户失败: %w", err)
	}
	if existingUser != nil {
		logger.Info("管理员用户已存在", zap.String("username", DefaultAdminUsername))
		return nil
	}

	userService := application.NewUserApplicationService(userRepo, idGen)
	user, err := userService.CreateUser(ctx, application.CreateUserCommand{
		Username:    DefaultAdminUsername,
		DisplayName: "系统管理员",
		Email:       "admin@local.dev",
		Password:    DefaultAdminPassword,
	})
	if err != nil {
		return fmt.Errorf("创建管理员用户失败: %w", err)
	}

	logger.Info("管理员用户创建成功",
		zap.String("username", user.Username()),
		zap.String("userCode", user.UserCode().String()),
	)
	fmt.Printf("初始密码: %s (请登录后立即修改)\n", DefaultAdminPassword)
	return nil
}

func runDeleteAdmin(logger *zap.Logger) error {
	userRepo, _, cleanup, err := getDBAndUserRepo(logger)
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := context.Background()
	existingUser, err := userRepo.FindByUsername(ctx, DefaultAdminUsername)
	if err != nil {
		return fmt.Errorf("查找用户失败: %w", err)
	}
	if existingUser == nil {
		logger.Info("管理员用户不存在", zap.String("username", DefaultAdminUsername))
		return nil
	}

	if err := userRepo.Delete(ctx, existingUser.ID()); err != nil {
		return fmt.Errorf("删除管理员用户失败: %w", err)
	}

	logger.Info("管理员用户已删除", zap.String("username", DefaultAdminUsername))
	return nil
}

func getDBAndUserRepo(logger *zap.Logger) (domain.UserRepository, domain.IDGenerator, func(), error) {
	dbPath := resolveDBPath()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("打开数据库失败(%s): %w", dbPath, err)
	}

	if err := _persistence.InitSchema(db); err != nil {
		db.Close()
		return nil, nil, nil, fmt.Errorf("初始化数据库结构失败(%s): %w", dbPath, err)
	}

	idGenerator := utils.NewNanoIDGenerator(utils.DefaultIDSize)

	userRepo := _persistence.NewSQLiteUserRepository(db)
	cleanup := func() {
		db.Close()
	}

	return userRepo, idGenerator, cleanup, nil
}
