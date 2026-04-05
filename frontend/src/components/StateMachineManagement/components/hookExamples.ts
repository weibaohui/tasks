/**
 * Hook 示例配置
 */

export interface HookExample {
  id: string;
  name: string;
  description: string;
  category: 'notification' | 'deployment' | 'approval' | 'custom';
  type: 'webhook' | 'command';
  config: {
    url?: string;
    method?: string;
    command?: string;
  };
  timeout?: number;
  retry?: number;
}

// 按场景分类的示例库
export const hookExamples: HookExample[] = [
  // 通知类
  {
    id: 'feishu-text',
    name: '飞书文本通知',
    description: '发送文本消息到飞书群机器人',
    category: 'notification',
    type: 'webhook',
    config: {
      url: 'https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-here',
      method: 'POST',
    },
    timeout: 30,
    retry: 1,
  },
  {
    id: 'feishu-card',
    name: '飞书卡片通知',
    description: '发送富文本卡片消息到飞书',
    category: 'notification',
    type: 'webhook',
    config: {
      url: 'https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-here',
      method: 'POST',
    },
    timeout: 30,
    retry: 1,
  },
  {
    id: 'dingtalk-text',
    name: '钉钉文本通知',
    description: '发送文本消息到钉钉群机器人',
    category: 'notification',
    type: 'webhook',
    config: {
      url: 'https://oapi.dingtalk.com/robot/send?access_token=your-token',
      method: 'POST',
    },
    timeout: 30,
    retry: 1,
  },
  {
    id: 'wecom-text',
    name: '企业微信文本通知',
    description: '发送文本消息到企业微信群机器人',
    category: 'notification',
    type: 'webhook',
    config: {
      url: 'https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=your-key',
      method: 'POST',
    },
    timeout: 30,
    retry: 1,
  },
  {
    id: 'slack-webhook',
    name: 'Slack Webhook',
    description: '发送消息到 Slack 频道',
    category: 'notification',
    type: 'webhook',
    config: {
      url: 'https://hooks.slack.com/services/xxx/yyy/zzz',
      method: 'POST',
    },
    timeout: 30,
    retry: 1,
  },

  // 部署类
  {
    id: 'deploy-script',
    name: '执行部署脚本',
    description: '在服务器上执行部署命令',
    category: 'deployment',
    type: 'command',
    config: {
      command: '/bin/bash /scripts/deploy.sh {{requirement_id}}',
    },
    timeout: 300,
    retry: 0,
  },
  {
    id: 'docker-compose',
    name: 'Docker Compose 部署',
    description: '使用 docker-compose 更新服务',
    category: 'deployment',
    type: 'command',
    config: {
      command: 'docker-compose -f /opt/app/docker-compose.yml pull && docker-compose -f /opt/app/docker-compose.yml up -d',
    },
    timeout: 300,
    retry: 0,
  },
  {
    id: 'kubectl-deploy',
    name: 'Kubernetes 部署',
    description: 'kubectl apply 部署应用到 K8s',
    category: 'deployment',
    type: 'command',
    config: {
      command: 'kubectl apply -f /k8s/deployment.yaml && kubectl rollout status deployment/app -n production',
    },
    timeout: 300,
    retry: 0,
  },

  // 审批类
  {
    id: 'approval-request',
    name: '发起审批请求',
    description: '调用审批系统创建审批流',
    category: 'approval',
    type: 'webhook',
    config: {
      url: 'https://your-approval-system.com/api/approvals',
      method: 'POST',
    },
    timeout: 60,
    retry: 2,
  },
  {
    id: 'send-email',
    name: '发送邮件',
    description: '触发邮件发送通知',
    category: 'approval',
    type: 'webhook',
    config: {
      url: 'https://your-email-service.com/api/send',
      method: 'POST',
    },
    timeout: 30,
    retry: 2,
  },

  // 自定义类
  {
    id: 'http-request',
    name: '通用 HTTP 请求',
    description: '发送 HTTP 请求到指定端点',
    category: 'custom',
    type: 'webhook',
    config: {
      url: 'https://your-api-endpoint.com/webhook',
      method: 'POST',
    },
    timeout: 60,
    retry: 1,
  },
  {
    id: 'shell-command',
    name: 'Shell 命令',
    description: '执行任意 shell 命令',
    category: 'custom',
    type: 'command',
    config: {
      command: '/bin/bash -c "your command here"',
    },
    timeout: 60,
    retry: 0,
  },
];

// 按分类分组的示例
export const examplesByCategory = {
  notification: hookExamples.filter((e) => e.category === 'notification'),
  deployment: hookExamples.filter((e) => e.category === 'deployment'),
  approval: hookExamples.filter((e) => e.category === 'approval'),
  custom: hookExamples.filter((e) => e.category === 'custom'),
};

// 分类名称映射
export const categoryNames = {
  notification: '通知',
  deployment: '部署',
  approval: '审批',
  custom: '自定义',
};
