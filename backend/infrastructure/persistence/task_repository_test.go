/**
 * SQLite 任务仓储集成测试
 */
package persistence

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/weibh/taskmanager/domain"
)

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// 使用内存数据库进行测试，速度快且每次干净
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("无法连接数据库: %v", err)
	}

	err = InitSchema(db)
	if err != nil {
		t.Fatalf("初始化 Schema 失败: %v", err)
	}

	cleanup := func() {
		db.Close()
	}

	return db, cleanup
}

func createTestTask() *domain.Task {
	task, _ := domain.NewTask(
		domain.NewTaskID("test-id-1"),
		domain.NewTraceID("trace-1"),
		domain.NewSpanID("span-1"),
		nil,
		"测试任务",
		"测试描述",
		domain.TaskTypeDataProcessing,
		map[string]interface{}{"key": "value"},
		60*time.Second,
		3,
		5,
	)
	return task
}

func TestSQLiteTaskRepository_SaveAndFind(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewSQLiteTaskRepository(db)
	ctx := context.Background()

	// 1. 保存任务
	task := createTestTask()
	err := repo.Save(ctx, task)
	if err != nil {
		t.Fatalf("保存任务失败: %v", err)
	}

	// 2. 查找任务
	foundTask, err := repo.FindByID(ctx, task.ID())
	if err != nil {
		t.Fatalf("查找任务失败: %v", err)
	}

	// 3. 验证字段
	if foundTask.ID() != task.ID() {
		t.Errorf("期望 ID 为 %s, 实际为 %s", task.ID(), foundTask.ID())
	}
	if foundTask.Name() != task.Name() {
		t.Errorf("期望名称为 %s, 实际为 %s", task.Name(), foundTask.Name())
	}
	if foundTask.Status() != task.Status() {
		t.Errorf("期望状态为 %v, 实际为 %v", task.Status(), foundTask.Status())
	}

	// 4. 更新任务并再次保存
	task.Start()
	task.UpdateProgress(100, 50, "处理中", "一半")
	err = repo.Save(ctx, task)
	if err != nil {
		t.Fatalf("更新任务失败: %v", err)
	}

	// 5. 再次查找验证更新
	updatedTask, err := repo.FindByID(ctx, task.ID())
	if err != nil {
		t.Fatalf("查找更新后任务失败: %v", err)
	}

	if updatedTask.Status() != domain.TaskStatusRunning {
		t.Errorf("期望状态为 Running, 实际为 %v", updatedTask.Status())
	}
	if updatedTask.Progress().Current() != 50 {
		t.Errorf("期望进度为 50, 实际为 %d", updatedTask.Progress().Current())
	}
}

func TestSQLiteTaskRepository_FindByTraceID(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewSQLiteTaskRepository(db)
	ctx := context.Background()

	task1 := createTestTask()
	repo.Save(ctx, task1)

	task2, _ := domain.NewTask(
		domain.NewTaskID("test-id-2"),
		domain.NewTraceID("trace-1"), // 相同的 TraceID
		domain.NewSpanID("span-2"),
		nil,
		"测试任务2",
		"",
		domain.TaskTypeDataProcessing,
		nil,
		60*time.Second,
		0,
		0,
	)
	repo.Save(ctx, task2)

	tasks, err := repo.FindByTraceID(ctx, domain.NewTraceID("trace-1"))
	if err != nil {
		t.Fatalf("根据 TraceID 查找任务失败: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("期望找到 2 个任务, 实际找到 %d 个", len(tasks))
	}
}

func TestSQLiteTaskRepository_DeleteAndExists(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := NewSQLiteTaskRepository(db)
	ctx := context.Background()

	task := createTestTask()
	repo.Save(ctx, task)

	// 测试 Exists
	exists, err := repo.Exists(ctx, task.ID())
	if err != nil {
		t.Fatalf("检查存在失败: %v", err)
	}
	if !exists {
		t.Error("期望任务存在")
	}

	// 测试 Delete
	err = repo.Delete(ctx, task.ID())
	if err != nil {
		t.Fatalf("删除任务失败: %v", err)
	}

	// 再次测试 Exists
	exists, err = repo.Exists(ctx, task.ID())
	if err != nil {
		t.Fatalf("检查存在失败: %v", err)
	}
	if exists {
		t.Error("期望任务已被删除")
	}
}

func TestMain(m *testing.M) {
	// 如果需要创建真实的持久化文件来测试，可以在这里处理
	os.Exit(m.Run())
}
