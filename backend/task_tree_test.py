#!/usr/bin/env python3
"""创建多层嵌套任务树测试"""
import json
import urllib.request
import time
import random

BASE_URL = "http://localhost:8888"

def create_task(name, task_type, parent_id=None, trace_id=None):
    """创建任务"""
    data = {
        "name": name,
        "type": task_type,
        "timeout": 60
    }
    if parent_id:
        data["parent_id"] = parent_id
    if trace_id:
        data["trace_id"] = trace_id

    req = urllib.request.Request(
        f"{BASE_URL}/api/v1/tasks",
        data=json.dumps(data).encode(),
        headers={"Content-Type": "application/json"}
    )
    try:
        with urllib.request.urlopen(req) as resp:
            return json.loads(resp.read())
    except Exception as e:
        print(f"  ⚠️ 创建失败: {e}")
        return None

def build_tree(parent_id, trace_id, depth, max_depth, prefix=""):
    """递归构建任务树"""
    if depth > max_depth:
        return

    if depth >= 3:
        width = random.randint(1, 2)
    else:
        width = random.randint(2, 4)

    task_types = ["data_processing", "file_operation", "api_call", "custom"]

    for i in range(1, width + 1):
        task_type = random.choice(task_types)
        is_leaf = (depth >= max_depth - 1) or (random.random() < 0.3)

        if is_leaf:
            name = f"{prefix}叶子-{depth}-{i}"
        else:
            name = f"{prefix}节点-{depth}-{i}"

        result = create_task(name, task_type, parent_id, trace_id)
        if result:
            task_id = result["id"]
            bar = "│" * (depth - 1) + ("├──" if depth > 0 else "")
            print(f"{prefix}{bar} [{task_type[:3]}] {name}")

            if depth < max_depth and not is_leaf:
                build_tree(task_id, trace_id, depth + 1, max_depth, prefix + "│   ")
            elif depth < max_depth:
                for j in range(random.randint(1, 3)):
                    leaf_name = f"{prefix}│   └── 叶子-{depth}-{i}-{j}"
                    leaf_type = random.choice(task_types)
                    leaf_result = create_task(leaf_name, leaf_type, task_id, trace_id)
                    if leaf_result:
                        print(f"{prefix}│   └── [{leaf_type[:3]}] {leaf_name}")

def main():
    print("=" * 50)
    print("TaskTree 压力测试 - 多层嵌套任务")
    print("=" * 50)

    root_name = f"🌳 根任务 {time.strftime('%H:%M:%S')}"
    print(f"\n创建根任务: {root_name}")

    root = create_task(root_name, "data_processing")
    if not root:
        print("❌ 创建根任务失败，请确保后端运行在 localhost:8888")
        return

    root_id = root["id"]
    root_trace = root["trace_id"]

    print(f"根任务 ID:    {root_id}")
    print(f"根任务 Trace: {root_trace}")
    print("\n📊 任务树结构:\n")

    build_tree(root_id, root_trace, 1, max_depth=4)

    print("\n" + "=" * 50)
    print("✅ 任务树生成完成!")
    print(f"🌐 http://localhost:3000/tasks/trace/{root_trace}/tree")
    print("=" * 50)

if __name__ == "__main__":
    main()
