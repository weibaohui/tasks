#!/bin/bash
# task_tree_test.sh - 创建多层嵌套任务树

BASE_URL="http://localhost:8888"
MAX_DEPTH=5
MIN_WIDTH=2
MAX_WIDTH=4

# 随机数
random_int() {
    echo $((RANDOM % ($2 - $1 + 1) + $1))
}

# 创建任务
create_task() {
    local name="$1"
    local task_type="$2"
    local parent_id="$3"
    local trace_id="$4"

    local data="{\"name\":\"$name\",\"type\":\"$task_type\",\"timeout\":60}"
    if [ -n "$parent_id" ]; then
        data=$(echo "$data" | jq --arg p "$parent_id" '. + {parent_id: $p}')
    fi
    if [ -n "$trace_id" ]; then
        data=$(echo "$data" | jq --arg t "$trace_id" '. + {trace_id: $t}')
    fi

    curl -s -X POST "$BASE_URL/api/v1/tasks" \
        -H "Content-Type: application/json" \
        -d "$data"
}

# 递归创建子任务
spawn_tasks() {
    local parent_id="$1"
    local trace_id="$2"
    local depth="$3"
    local prefix="$4"

    if [ "$depth" -gt "$MAX_DEPTH" ]; then
        return
    fi

    # 随机子任务数量
    if [ "$depth" -ge 3 ]; then
        width=$(random_int 1 2)
    else
        width=$(random_int $MIN_WIDTH $MAX_WIDTH)
    fi

    local task_types=("data_processing" "file_operation" "api_call" "custom")
    local type_idx=0

    for i in $(seq 1 $width); do
        type_idx=$((RANDOM % 4))
        task_type="${task_types[$type_idx]}"

        # 叶子节点使用特殊名称
        if [ "$depth" -ge $((MAX_DEPTH - 1)) ] || [ $((RANDOM % 100)) -lt 30 ]; then
            name="${prefix}叶子-${depth}-${i}"
        else
            name="${prefix}节点-${depth}-${i}"
        fi

        # 创建任务
        result=$(create_task "$name" "$task_type" "$parent_id" "$trace_id")
        task_id=$(echo "$result" | jq -r '.id')

        if [ -n "$task_id" ] && [ "$task_id" != "null" ]; then
            echo "${prefix}├── [${task_type}] ${name} -> ${task_id:0:12}"

            # 递归创建子任务
            if [ "$depth" -lt "$MAX_DEPTH" ] && [ $((RANDOM % 100)) -lt 70 ]; then
                spawn_tasks "$task_id" "$trace_id" $((depth + 1)) "${prefix}│   "
            fi
        fi
    done
}

# 主函数
main() {
    echo "╔══════════════════════════════════════════════╗"
    echo "║       TaskTree 压力测试                     ║"
    echo "╚══════════════════════════════════════════════╝"
    echo "配置: MAX_DEPTH=$MAX_DEPTH, MIN_WIDTH=$MIN_WIDTH, MAX_WIDTH=$MAX_WIDTH"
    echo ""

    # 创建根任务
    root_name="🌳 根任务-$(date '+%H:%M:%S')"
    echo "创建根任务: $root_name"
    root_result=$(create_task "$root_name" "data_processing" "" "")
    root_id=$(echo "$root_result" | jq -r '.id')
    root_trace=$(echo "$root_result" | jq -r '.trace_id')

    if [ -z "$root_id" ] || [ "$root_id" == "null" ]; then
        echo "❌ 创建根任务失败"
        exit 1
    fi

    echo ""
    echo "🌱 根任务已创建:"
    echo "   ID:     $root_id"
    echo "   Trace:  $root_trace"
    echo ""
    echo "📊 开始生成任务树..."
    echo ""

    # 递归生成子任务
    total=$(spawn_tasks "$root_id" "$root_trace" 1 "")

    echo ""
    echo "✅ 任务树生成完成!"
    echo ""
    echo "🌐 查看任务树: http://localhost:3000/tasks/trace/${root_trace}/tree"
}

main
