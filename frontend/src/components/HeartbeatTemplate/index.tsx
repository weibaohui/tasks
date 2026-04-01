/**
 * 心跳模板编辑器组件
 */
import React from 'react';
import { Form, Button, Input, Space, Tooltip } from 'antd';
import { CopyOutlined, FileTextOutlined, ReloadOutlined } from '@ant-design/icons';

const { TextArea } = Input;

export const DEFAULT_HEARTBEAT_TEMPLATE = `你是一个心跳，你有两个任务：
1. 需求派发。
2. 处理PR。

# 任务一：派发需求
## 1.1 查看需求列表
使用taskmanager requirement list命令查看当前未处理的需求列表，找到状态为status=todo，requirement_type=normal的需求，并按创建时间排序。
## 1.2 派发需求
派发第一个待处理的需求。

## 1.3. 派发注意事项
- 只派发 todo 状态的需求
- 已完成、进行中、失败的需求不要派发
- 每次心跳最多派发一个需求
- 优先派发最早创建的需求


# 任务二：处理PR

## 1. 获取待合并的PR列表
gh pr list --state open --mergeable non-conflicting --json number,title,author,body,url

## 2. 分析每个PR
对于每个待合并的PR：
1. 对于所有评论已解决、CI通过、代码审查通过的PR，可以评论 /lgtm。使用gh pr comment <PR_NUMBER> --body "/lgtm"。
2. 对于已经有 /lgtm 的评论，可以直接合并到main分支，并删除源分支
3. 你判断reviewer提出的评论建议是否需要修复，如需修复，请创建需求让另一AI执行修复；如不需要修复，直接评论 /lgtm。注意你不要自己修复。
4. 创建代码修复需求
使用taskmanager requirement create --project-id <PROJECT_ID> --title "[修复] <修复标题>" --description "## 背景来源：PR #<PR号> https://github.com/owner/repo/pull/<PR号> 评论：<reviewer评论内容及摘要> ## 修复分支 请进入<branch_name>进行修复，修复完成后提交并推送该分支。" --acceptance "具体验收标准"

## 3.PR处理重要原则
1. 需求必须独立可完成
2. 描述要让AI无需再看PR就能工作
3. 代码修复要写清在哪个分支、哪个文件、具体修复内容
4. 仅在必要时创建需求
5. 没问题的PR写 /lgtm`;

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
