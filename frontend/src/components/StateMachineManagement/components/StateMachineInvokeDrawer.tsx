/**
 * StateMachine Invoke Drawer Component
 * Shows how to invoke a state machine via HTTP, CLI, and SDK
 */
import React, { useState } from 'react';
import { Drawer, Typography, Tabs, Card, Space, Tag, Button, message, Table, Alert } from 'antd';
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

  const httpCode = `curl -X POST "${apiBaseUrl}/requirements/{requirement-id}/transitions" \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer <your-token>" \\
  -d '{
    "trigger": "start"
  }'`;

  const cliCode = `# ============================================
# 通用状态机 CLI 使用指南（纯规则引擎，无业务绑定）
# ============================================

# 1. 列出所有状态机模板
taskmanager statemachine list

# 2. 获取状态机规则详情
taskmanager statemachine get --machine=${stateMachine.name}

# 3. 查询指定状态的可用触发器
taskmanager statemachine triggers --machine=${stateMachine.name} --from=${stateMachine.config.initial_state}

# 4. 验证状态转换是否允许（从A到B）
taskmanager statemachine validate --machine=${stateMachine.name} --from=${stateMachine.config.initial_state} --to=<目标状态>

# 5. 执行状态转换（返回目标状态）
taskmanager statemachine execute --machine=${stateMachine.name} --from=${stateMachine.config.initial_state} --trigger=<触发器>`;

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
          <Alert
            message="纯通用状态机引擎"
            description="CLI 命令只负责状态流转规则计算，不管理业务实例ID。业务层自行管理实例和状态存储。"
            type="info"
            showIcon
          />

          <Card
            title="完整使用示例"
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
            <pre style={{ margin: 0, overflow: 'auto', fontSize: 12 }}>
              <code>{cliCode}</code>
            </pre>
          </Card>

          <Card title="常用命令速查">
            <Space direction="vertical" style={{ width: '100%' }}>
              <div>
                <Text strong>
                  <Tag color="blue">list</Tag>
                </Text>
                <Text code>taskmanager statemachine list</Text>
                <Text type="secondary">列出所有状态机模板</Text>
              </div>
              <div>
                <Text strong>
                  <Tag color="blue">get</Tag>
                </Text>
                <Text code>taskmanager statemachine get --machine=&lt;name&gt;</Text>
                <Text type="secondary">获取状态机规则详情</Text>
              </div>
              <div>
                <Text strong>
                  <Tag color="blue">triggers</Tag>
                </Text>
                <Text code>taskmanager statemachine triggers -m &lt;name&gt; -f &lt;state&gt;</Text>
                <Text type="secondary">查询状态的可用触发器</Text>
              </div>
              <div>
                <Text strong>
                  <Tag color="green">validate</Tag>
                </Text>
                <Text code>taskmanager statemachine validate -m &lt;name&gt; -f &lt;from&gt; -t &lt;to&gt;</Text>
                <Text type="secondary">验证从A到B是否允许</Text>
              </div>
              <div>
                <Text strong>
                  <Tag color="green">execute</Tag>
                </Text>
                <Text code>taskmanager statemachine execute -m &lt;name&gt; -f &lt;state&gt; -t &lt;trigger&gt;</Text>
                <Text type="secondary">执行状态转换</Text>
              </div>
            </Space>
          </Card>

          <Card title="参数说明">
            <Space direction="vertical">
              <Text>
                <Tag>--machine / -m</Tag> 状态机模板名称（如：{stateMachine.name}）
              </Text>
              <Text>
                <Tag>--from / -f</Tag> 源状态（当前状态）
              </Text>
              <Text>
                <Tag>--to / -t</Tag> 目标状态（validate命令使用）
              </Text>
              <Text>
                <Tag>--trigger / -t</Tag> 触发器名称（execute命令使用）
              </Text>
            </Space>
          </Card>

          <Card title="典型工作流示例">
            <pre style={{ margin: 0, fontSize: 12, background: '#f6ffed', padding: 12, borderRadius: 4 }}>
              <code>{`# 1. 查看当前状态有哪些可用触发器
taskmanager statemachine triggers -m "${stateMachine.name}" -f "${stateMachine.config.initial_state}"

# 2. 验证转换是否合法
taskmanager statemachine validate -m "${stateMachine.name}" -f "${stateMachine.config.initial_state}" -t "<目标状态>"

# 3. 执行转换（业务层自行保存结果）
RESULT=$(taskmanager statemachine execute -m "${stateMachine.name}" -f "${stateMachine.config.initial_state}" -t "<触发器>")
echo "转换结果: $RESULT"`}</code>
            </pre>
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
            title="触发状态转换"
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
                <Tag>requirement-id</Tag> 需求ID，状态转换的目标对象
              </Text>
            </Space>
          </Card>

          <Card title="其他 API">
            <Space direction="vertical">
              <Text>
                <Tag>GET</Tag>
                <Text code>/state-machines</Text>
                列出所有状态机
              </Text>
              <Text>
                <Tag>GET</Tag>
                <Text code>/state-machines/{'{id}'}</Text>
                获取状态机详情
              </Text>
              <Text>
                <Tag>GET</Tag>
                <Text code>/requirements/{'{id}'}/state</Text>
                获取需求当前状态
              </Text>
              <Text>
                <Tag>GET</Tag>
                <Text code>/requirements/{'{id}'}/transitions/history</Text>
                获取转换历史
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
      width={850}
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
          <Paragraph>
            <Text strong>状态数: </Text>
            <Tag>{stateMachine.config.states.length}</Tag>
            <Text strong style={{ marginLeft: 16 }}>转换规则数: </Text>
            <Tag>{stateMachine.config.transitions.length}</Tag>
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
