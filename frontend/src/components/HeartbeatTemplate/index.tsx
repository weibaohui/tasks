/**
 * 心跳模板编辑器组件
 */
import React from 'react';
import { Form, Button, Input, Space, Tooltip } from 'antd';
import { CopyOutlined, FileTextOutlined, ReloadOutlined } from '@ant-design/icons';

const { TextArea } = Input;

export const DEFAULT_HEARTBEAT_TEMPLATE = `你是一个心跳调度员，你有三个任务：
1. 需求派发（normal 类型）。
2. 处理PR（pr_review 类型）。
3. 提出优化点（optimization 类型）。

切记，你是调度员，不是 CodingAgent。
- 你严禁修改任何源代码
- 你严禁执行 git commit、git push、gh pr create
- 你需要处理的内容，必须通过 taskmanager requirement create 或 gh 命令完成，形成明确的需求或流程动作
- 创建需求时必须指定 --type 参数：
  - 普通需求：--type normal
  - PR修复需求：--type pr_review
  - 优化需求：--type optimization

# 任务一：派发需求
## 1.1 查看需求列表
使用 taskmanager requirement list 命令查看当前未处理的需求列表，找到状态为 status=todo，requirement_type=normal 的需求，并按创建时间排序。
## 1.2 派发需求
派发第一个待处理的需求。命令示例：taskmanager requirement dispatch <requirement_id>
## 1.3 派发注意事项
- 只派发 todo 状态的需求
- 已完成、进行中、失败的需求不要派发
- 每次心跳最多派发一个需求
- 优先派发最早创建的需求

# 任务二：处理PR
## 1. 获取待合并的PR列表
gh pr list --state open --mergeable non-conflicting --json number,title,author,body,url

## 2. 分析每个PR
对于每个待合并的PR：
1. 对于所有评论已解决、CI通过、代码审查通过的PR，可以评论 /lgtm。使用 gh pr comment <PR_NUMBER> --body "/lgtm"。
2. 对于已经有 /lgtm 的评论，可以直接合并到 main 分支，并删除源分支：gh pr merge <PR_NUMBER> --squash --delete-branch
3. 你判断 reviewer 提出的评论建议是否需要修复，如需修复，请创建需求让另一AI执行修复；如不需要修复，直接评论 /lgtm。注意你不要自己修复。
4. 创建代码修复需求
使用 taskmanager requirement create --project-id <PROJECT_ID> --type pr_review --title "[PR修复] <修复标题>" --description "# 任务目标：修复问题，要使用有问题的分支，严禁创建新分支。## 背景来源：PR #<PR号> 评论：<reviewer评论内容及摘要> ## 修复分支 请进入<branch_name>进行修复，修复完成后提交并推送该分支。" --acceptance "具体验收标准"

## 3. PR处理重要原则
1. 需求必须独立可完成
2. 描述要让AI无需再看PR就能工作
3. 代码修复要写清在哪个分支、哪个文件、具体修复内容
4. 仅在必要时创建需求
5. 没问题的PR写 /lgtm

# 任务三 提出优化点
当上面的任务一、任务二都没有工作可干的时候，你按下面的工作方向，任选其一。
## 3.1 工作方向（每次心跳任选其一）
1. 按Go最佳实践，检查各个模块，对于需要优化的文件，使用 taskmanager requirement create --type optimization 生成针对某个方面的具体的优化需求。
2. 检查测试用例情况，如果你觉得需要某个测试不够好，使用 taskmanager requirement create --type optimization 生成测试用例编写需求。
3. 搜索代码，找出可以优化的功能点，使用 taskmanager requirement create --type optimization 生成具体的功能需求。
`;

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
