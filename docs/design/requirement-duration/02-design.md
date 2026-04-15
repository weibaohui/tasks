# 需求列表展示耗时 - 设计说明

## 方案概述

在前端需求列表页面（`ProjectRequirementPage`）的表格列配置中新增“耗时”列：

- 通过 `Requirement.started_at/completed_at` 或 `Requirement.agent_runtime.started_at/ended_at` 计算毫秒耗时
- 使用 `humanize-duration` 输出紧凑格式字符串
- 不改动后端 API 与数据结构

## 前端改动点

### 依赖

- 新增依赖：`humanize-duration`

### 代码位置

- 页面：`frontend/src/pages/ProjectRequirementPage.tsx`
  - 在 `requirementColumns` 中插入新列
  - 增加两个纯函数：
    - 计算耗时毫秒：从数据对象取时间并做兜底
    - 格式化耗时：将毫秒输出为紧凑格式

## 人性化时长格式

通过 `humanize-duration` 的 `humanizer` 自定义语言与输出规则：

- `spacer: ''`：数字与单位无空格
- `delimiter: ''`：多个单位间无分隔符
- `largest: 2`：最多显示两个单位（如 `2d5h`、`1h28m`）
- `units: ['d', 'h', 'm', 's']`：限定输出单位集合
- 自定义语言：单位缩写为 `d/h/m/s`

