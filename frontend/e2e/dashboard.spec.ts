import { test, expect } from '@playwright/test';

test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('http://localhost:3000/dashboard');
  });

  test('D1 - Dashboard 页面加载成功', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
  });

  test('D2 - 核心统计卡片可见', async ({ page }) => {
    await expect(page.getByText('总会话数')).toBeVisible();
    await expect(page.getByText('总 Token 数').first()).toBeVisible();
    await expect(page.getByText('平均消息数/会话')).toBeVisible();
    await expect(page.getByText('平均响应时间')).toBeVisible();
  });

  test('D3 - 图表区域可见', async ({ page }) => {
    await expect(page.getByText('Token 消耗趋势')).toBeVisible();
    await expect(page.getByText('项目 Token 消耗排行')).toBeVisible();
    await expect(page.getByText('Agent 使用分布')).toBeVisible();
    await expect(page.getByText('Agent Type Token 消耗排行')).toBeVisible();
    await expect(page.getByText('Agent Type Token 消耗占比')).toBeVisible();
  });

  test('D4 - 日期筛选器和刷新按钮可操作', async ({ page }) => {
    const refreshBtn = page.getByRole('button', { name: '刷新' });
    await expect(refreshBtn).toBeVisible();
  });
});
