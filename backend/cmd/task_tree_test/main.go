/**
 * TaskTreeTest - 任务树压力测试
 * 创建一个多层嵌套的任务树，模拟真实场景
 *
 * 使用方法:
 *   go run cmd/task_tree_test/main.go
 *
 * 或者在 tests 中运行:
 *   go test -v -run TestTaskTreeStress
 */
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/weibh/taskmanager/infrastructure/utils"
)

const (
	BaseURL   = "http://localhost:8888"
	MaxDepth  = 5 // 最大深度
	MaxWidth  = 5 // 每层最大子任务数
	MinWidth  = 2 // 每层最小子任务数
	TaskTypes = "data_processing,file_operation,api_call,custom"
)

var (
	httpClient = &http.Client{Timeout: 10 * time.Second}
	idGen      = utils.NewNanoIDGenerator(21)
	taskCount  int
	totalDepth int
)

type CreateTaskRequest struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	ParentID string                 `json:"parent_id,omitempty"`
	TraceID  string                 `json:"trace_id,omitempty"`
	Metadata map[string]interface{} `json:"metadata"`
	Timeout  int64                  `json:"timeout"`
	Priority int                    `json:"priority"`
}

type CreateTaskResponse struct {
	ID       string `json:"id"`
	TraceID  string `json:"trace_id"`
	SpanID   string `json:"span_id"`
	ParentID string `json:"parent_id,omitempty"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Status   string `json:"status"`
}

func main() {
	rand.Seed(time.Now().UnixNano())

	fmt.Println("╔══════════════════════════════════════════════╗")
	fmt.Println("║       TaskTree Stress Test                  ║")
	fmt.Println("╚══════════════════════════════════════════════╝")
	fmt.Printf("配置: MaxDepth=%d, MinWidth=%d, MaxWidth=%d\n\n", MaxDepth, MinWidth, MaxWidth)

	// 创建根任务
	rootName := fmt.Sprintf("🌳 根任务 [%s]", time.Now().Format("15:04:05"))
	root := createTask(rootName, "data_processing", "", "")
	if root == nil {
		fmt.Println("❌ 创建根任务失败")
		os.Exit(1)
	}

	fmt.Printf("\n🌱 根任务已创建: %s (ID: %s, Trace: %s)\n", root.Name, root.ID, root.TraceID)
	fmt.Println("\n📊 开始生成任务树...")

	// 从根任务开始递归创建子任务
	depth := 1
	spawnSubTasks(root.ID, root.TraceID, depth)

	fmt.Printf("\n✅ 任务树生成完成!\n")
	fmt.Printf("📊 总任务数: %d\n", taskCount)
	fmt.Printf("📊 最大深度: %d\n", totalDepth)
	fmt.Printf("🌐 访问: http://localhost:3000/tasks/trace/%s/tree\n", root.TraceID)
}

func spawnSubTasks(parentID, traceID string, depth int) {
	if depth > MaxDepth {
		return
	}

	// 随机决定这层的子任务数量
	width := rand.Intn(MaxWidth-MinWidth+1) + MinWidth

	// 模拟真实场景：越深层子任务越少
	if depth >= 3 {
		width = rand.Intn(3) + 1
	}

	taskTypes := []string{"data_processing", "file_operation", "api_call", "custom"}

	for i := 0; i < width; i++ {
		// 随机决定是否继续往下
		if depth < MaxDepth && rand.Float32() < 0.7 {
			// 有子任务
			name := generateTaskName(depth, i)
			child := createTask(name, taskTypes[rand.Intn(len(taskTypes))], parentID, traceID)
			if child == nil {
				continue
			}
			indent := indentStr(depth)
			fmt.Printf("%s├── [%s] %s (子任务: %d个)\n", indent, child.Type, child.Name, rand.Intn(MaxWidth)+1)

			// 递归创建子任务
			if depth < MaxDepth-1 && rand.Float32() < 0.8 {
				spawnSubTasks(child.ID, traceID, depth+1)
			}
		} else {
			// 叶子任务
			name := generateLeafName(depth, i)
			leaf := createTask(name, taskTypes[rand.Intn(len(taskTypes))], parentID, traceID)
			if leaf == nil {
				continue
			}
			indent := indentStr(depth)
			fmt.Printf("%s└── [%s] %s\n", indent, leaf.Type, leaf.Name)
		}

		// 更新最大深度
		if depth > totalDepth {
			totalDepth = depth
		}

		// 添加随机延迟模拟真实场景
		time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
	}
}

func generateTaskName(depth, index int) string {
	prefixes := []string{"📂 处理", "📥 采集", "⚙️ 计算", "🔄 转换", "📤 导出", "🔍 分析", "💾 存储"}
	return fmt.Sprintf("%s 深度%d-%d [%s]", prefixes[rand.Intn(len(prefixes))], depth, index, idGen.Generate()[:8])
}

func generateLeafName(depth, index int) string {
	prefixes := []string{"✓ 完成", "→ 转发", "◆ 记录", "● 标记", "△ 确认"}
	return fmt.Sprintf("%s 叶子%d-%d [%s]", prefixes[rand.Intn(len(prefixes))], depth, index, idGen.Generate()[:8])
}

func indentStr(depth int) string {
	result := ""
	for i := 0; i < depth; i++ {
		result += "    "
	}
	return result
}

func createTask(name, taskType, parentID, traceID string) *CreateTaskResponse {
	taskCount++

	req := CreateTaskRequest{
		Name:     name,
		Type:     taskType,
		ParentID: parentID,
		TraceID:  traceID,
		Metadata: map[string]interface{}{
			"depth":     getDepthFromName(name),
			"index":     taskCount,
			"generated": time.Now().Unix(),
		},
		Timeout:  60000,
		Priority: rand.Intn(10),
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, "POST", BaseURL+"/api/v1/tasks", bytes.NewBuffer(body))
	if err != nil {
		return nil
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil
	}

	var result CreateTaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil
	}

	return &result
}

func getDepthFromName(name string) int {
	// 从名称中提取深度
	var depth int
	for _, c := range name {
		if c >= '0' && c <= '9' {
			depth = int(c - '0')
		}
	}
	return depth
}
