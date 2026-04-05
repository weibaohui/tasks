/**
 * StateMachine Invoke Drawer Component
 * Shows how to invoke a state machine via HTTP, CLI, and SDK
 */
import React, { useState } from 'react';
import { Drawer, Typography, Tabs, Card, Space, Tag, Button, message, Table } from 'antd';
import { CopyOutlined, ApiOutlined, CodeOutlined } from '@ant-design/icons';
import type { StateMachine } from '../../../types/stateMachine';

const { Title, Text, Paragraph } = Typography;

interface StateMachineInvokeDrawerProps {
  open: boolean;
  stateMachine: StateMachine | null;
  onClose: () => void;
}

export const StateMachineInvokeDrawer: React.FC<StateMachineInvokeDrawerProps> = ({
  open,
  stateMachine,
  onClose,
}) => {
  const [activeTab, setActiveTab] = useState('cli');

  if (!stateMachine) return null;

  const baseUrl = window.location.origin;
  const apiBaseUrl = `${baseUrl}/api/v1`;

  const httpCode = `curl -X POST "${apiBaseUrl}/state-machines/${stateMachine.id}/execute" \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer <your-token>" \\
  -d '{
    "trigger": "start",
    "context": {
      "requirement_id": "your-requirement-id",
      "project_id": "your-project-id"
    }
  }'`;

  const cliCode = `# 命令格式
taskmanager state-machine execute <state-machine-id> \\
  --trigger <trigger-name> \\
  --context '<json-context>'

# 示例 - context 支持多个属性
taskmanager state-machine execute ${stateMachine.id} \\
  --trigger start \\
  --context '{
    "requirement_id": "req-123",
    "project_id": "proj-456",
    "user_id": "user-789"
  }'`;

  const sdkCode = `package main

import (
    "context"
    "fmt"
    "log"

    tm "github.com/weibaohui/taskmanager/sdk/go"
)

func main() {
    // 创建客户端
    client := tm.NewClient("${baseUrl}")

    // 执行状态机
    resp, err := client.ExecuteStateMachine(context.Background(), "${stateMachine.id}", tm.ExecuteRequest{
        Trigger: "start",
        Context: map[string]interface{}{
            "requirement_id": "your-requirement-id",
            "project_id":     "your-project-id",
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Execution ID: %s\\n", resp.ExecutionID)
    fmt.Printf("Current State: %s\\n", resp.CurrentState)
}`;

  const handleCopy = (code: string) => {
    navigator.clipboard.writeText(code);
    message.success('已复制到剪贴板');
  };

  const items = [
    {
      key: 'cli',
      label: (
        <Space>
          <CodeOutlined />
          CLI
        </Space>
      ),
      children: (
        <Space direction="vertical" style={{ width: '100%' }} size="large">
          <Card
            title="命令行执行"
            extra={
              <Button
                icon={<CopyOutlined />}
                size="small"
                onClick={() => handleCopy(cliCode)}
              >
                复制
              </Button>
            }
          >
            <pre style={{ margin: 0, overflow: 'auto' }}>
              <code>{cliCode}</code>
            </pre>
          </Card>

          <Card title="常用命令">
            <Space direction="vertical">
              <Text>
                <Tag>taskmanager state-machine list</Tag> 列出所有状态机
              </Text>
              <Text>
                <Tag>taskmanager state-machine get {'<id>'}</Tag> 获取状态机详情
              </Text>
              <Text>
                <Tag>taskmanager state-machine execute {'<id>'}</Tag> 执行状态机
              </Text>
            </Space>
          </Card>
        </Space>
      ),
    },
    {
      key: 'http',
      label: (
        <Space>
          <ApiOutlined />
          HTTP
        </Space>
      ),
      children: (
        <Space direction="vertical" style={{ width: '100%' }} size="large">
          <Card
            title="执行状态机"
            extra={
              <Button
                icon={<CopyOutlined />}
                size="small"
                onClick={() => handleCopy(httpCode)}
              >
                复制
              </Button>
            }
          >
            <pre style={{ margin: 0, overflow: 'auto' }}>
              <code>{httpCode}</code>
            </pre>
          </Card>

          <Card title="认证说明">
            <Space direction="vertical">
              <Text>
                <Tag>Authorization</Tag> 使用 Bearer Token 认证，在请求头中添加：
              </Text>
              <Text code copyable>
                Authorization: Bearer &lt;your-token&gt;
              </Text>
            </Space>
          </Card>

          <Card title="参数说明">
            <Space direction="vertical">
              <Text>
                <Tag>trigger</Tag> 触发器名称，对应状态机配置中的 trigger
              </Text>
              <Text>
                <Tag>context</Tag> 执行上下文，可包含 requirement_id、project_id 等变量
              </Text>
            </Space>
          </Card>
        </Space>
      ),
    },
    {
      key: 'sdk',
      label: (
        <Space>
          <CodeOutlined />
          SDK
        </Space>
      ),
      children: (
        <Space direction="vertical" style={{ width: '100%' }} size="large">
          <Card
            title="Go SDK 示例"
            extra={
              <Button
                icon={<CopyOutlined />}
                size="small"
                onClick={() => handleCopy(sdkCode)}
              >
                复制
              </Button>
            }
          >
            <pre style={{ margin: 0, overflow: 'auto' }}>
              <code>{sdkCode}</code>
            </pre>
          </Card>

          <Card title="安装 SDK">
            <pre style={{ margin: 0 }}>
              <code>go get github.com/weibaohui/taskmanager/sdk/go</code>
            </pre>
          </Card>
        </Space>
      ),
    },
  ];

  return (
    <Drawer
      title={
        <Space>
          <span>调用状态机</span>
          {stateMachine && (
            <Tag color="blue">{stateMachine.name}</Tag>
          )}
        </Space>
      }
      placement="right"
      width={800}
      onClose={onClose}
      open={open}
    >
      <Space direction="vertical" style={{ width: '100%' }} size="large">
        <div>
          <Title level={5}>状态机信息</Title>
          <Paragraph>
            <Text strong>ID: </Text>
            <Text copyable>{stateMachine.id}</Text>
          </Paragraph>
          <Paragraph>
            <Text strong>名称: </Text>
            <Text>{stateMachine.name}</Text>
          </Paragraph>
          <Paragraph>
            <Text strong>初始状态: </Text>
            <Tag color="green">
              {stateMachine.config.initial_state}
            </Tag>
          </Paragraph>
          <div style={{ marginTop: 16 }}>
            <Text strong>可用触发器:</Text>
            <Table
              size="small"
              pagination={false}
              style={{ marginTop: 8 }}
              dataSource={stateMachine.config.transitions}
              columns={[
                {
                  title: '触发器',
                  dataIndex: 'trigger',
                  key: 'trigger',
                  width: 120,
                },
                {
                  title: '状态流转',
                  key: 'transition',
                  width: 200,
                  render: (_, record) => (
                    <span>{record.from} → {record.to}</span>
                  ),
                },
                {
                  title: '说明',
                  dataIndex: 'description',
                  key: 'description',
                  ellipsis: true,
                },
              ]}
            />
          </div>
        </div>

        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={items}
        />
      </Space>
    </Drawer>
  );
};
