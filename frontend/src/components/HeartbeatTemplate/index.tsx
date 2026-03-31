/**
 * 心跳模板编辑器组件
 */
import React from 'react';
import { Form, Button, Input, Space, Tooltip } from 'antd';
import { CopyOutlined, FileTextOutlined, ReloadOutlined } from '@ant-design/icons';

const { TextArea } = Input;

export const DEFAULT_HEARTBEAT_TEMPLATE = `# 心跳模板：PR 审查与需求生成

## 任务目标

检查项目的待处理 PR，分析评论内容，判断是否需要生成新的项目需求。

---

## 执行步骤

### 1. 获取待合并的 PR 列表

\`\`\`bash
# 查看所有待合并的PR
gh pr list --state open --mergeable non-conflicting --json number,title,author,body,url

# 查看PR详情和评论
gh pr view <PR_NUMBER> --json title,body,author,state,comments
\`\`\`

### 2. 分析每个 PR

对于每个待合并的 PR：

1. **检查 PR 描述**：确认是否包含明确的实施意图
2. **分析评论**：特别是代码审查评论，判断是否有以下情况：
   - reviewer 提出了明确的修改建议且开发者已确认
   - 有明确的 blocker 或 blocking 标记
   - reviewer 建议需要单独跟踪问题

### 3. 判断是否需要生成需求

**需要生成新需求的情况**：
- reviewer 提出的问题需要解决
- 需要请专业领域代码评审（如数据库设计、算法优化等）

**需求颗粒度**
- 一个需求为一个AI在本需求的描述下就可以独立完成的任务。拆分的需求之间不能产生互相的依赖。

**可以直接合并的情况**：
- 所有评论都已解决
- 没有需要进一步讨论的话题
- CI/CD 检查全部通过
- 代码审查已通过

### 4. 创建需求（如需要）

#### 4.1 代码修复需求

如果 PR 评论中要求代码修改，创建需求让另一 AI 执行修复：

\`\`\`bash
taskmanager requirement create \\
  --project-id <PROJECT_ID> \\
  --title "[修复] <修复标题>" \\
  --description "## 背景
来源：PR #<PR号> https://github.com/owner/repo/pull/<PR号>
评论：<reviewer评论摘要>

## 修复分支
<branch_name>

## 修复文件
<文件路径>

## 修复内容
<reviewer要求的代码修改详情>

## 修复步骤
<reviewer建议的修复步骤>

## 验收标准
1. <明确可测试的验收条件>
2. <修复后PR可合并>" \\
  --acceptance "1. <具体验收条件>"
\`\`\`

#### 4.2 新功能需求

\`\`\`bash
taskmanager requirement create \\
  --project-id <PROJECT_ID> \\
  --title "[功能] <功能标题>" \\
  --description "## 背景
来源：PR #<PR号> reviewer建议

## 任务
<reviewer建议的功能描述>

## 技术要求
<具体技术实现要求>

## 验收标准
<明确可测试的条件>" \\
  --acceptance "<具体验收条件>"
\`\`\`

### 5. 处理无需创建需求的 PR

如果 PR 可以合并，直接评论 \`/lgtm\`：

\`\`\`bash
# 评论 lgtm
gh pr comment <PR_NUMBER> --body "/lgtm"
\`\`\`

## 模板变量

| 变量 | 说明 |
|------|------|
| \`\${project.id}\` | 项目ID |
| \`\${project.name}\` | 项目名称 |
| \`\${project.git_repo_url}\` | Git仓库URL |
| \`\${project.default_branch}\` | 默认分支 |
| \`\${timestamp}\` | 执行时间戳 |

---

## 重要原则

1. **需求独立性**：每个需求必须能独立完成，不能依赖其他需求的结果
2. **描述完整性**：需求描述要让下一个AI无需再看PR就能开始工作
3. **代码修复要详细**：必须写清在哪个分支、哪个文件、哪行、具体修改什么
4. **验收明确**：每个需求必须有明确可测试的验收标准
5. **最小粒度**：如果一个PR涉及多个独立任务，创建多个需求
6. **仅在必要时创建**：不是所有PR评论都需要创建需求，只有需要后续跟踪的才创建
7. **没问题的PR**：直接评论 \`/lgtm\`，无需创建任何需求`;

interface HeartbeatTemplateEditorProps {
  value?: string;
  onChange?: (value: string) => void;
  disabled?: boolean;
}

export const HeartbeatTemplateEditor: React.FC<HeartbeatTemplateEditorProps> = ({
  value,
  onChange,
  disabled = false,
}) => {
  const [form] = Form.useForm();

  const handleFillTemplate = () => {
    form.setFieldsValue({ template: DEFAULT_HEARTBEAT_TEMPLATE });
    onChange?.(DEFAULT_HEARTBEAT_TEMPLATE);
  };

  const handleCopyTemplate = () => {
    navigator.clipboard.writeText(DEFAULT_HEARTBEAT_TEMPLATE);
  };

  const handleValuesChange = (_: unknown, allValues: { template?: string }) => {
    onChange?.(allValues.template || '');
  };

  return (
    <Form
      form={form}
      layout="vertical"
      onValuesChange={handleValuesChange}
      initialValues={{ template: value || DEFAULT_HEARTBEAT_TEMPLATE }}
    >
      <Form.Item
        name="template"
        label={
          <Space>
            <FileTextOutlined />
            <span>心跳模板</span>
          </Space>
        }
        extra="使用模板变量：\${project.id}, \${project.name}, \${project.git_repo_url}, \${project.default_branch}, \${timestamp}"
      >
        <TextArea
          rows={20}
          style={{ fontFamily: 'monospace' }}
          disabled={disabled}
          placeholder="输入心跳模板内容..."
        />
      </Form.Item>

      <Space style={{ marginTop: 8 }}>
        <Tooltip title="使用默认模板填充">
          <Button
            icon={<ReloadOutlined />}
            onClick={handleFillTemplate}
            disabled={disabled}
          >
            使用默认模板
          </Button>
        </Tooltip>
        <Tooltip title="复制默认模板到剪贴板">
          <Button
            icon={<CopyOutlined />}
            onClick={handleCopyTemplate}
            disabled={disabled}
          >
            复制模板
          </Button>
        </Tooltip>
      </Space>
    </Form>
  );
};
