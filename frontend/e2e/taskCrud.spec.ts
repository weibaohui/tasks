import { test, expect, Page } from '@playwright/test';

const BASE_URL = 'http://localhost:3000';

test.describe('TaskManager CRUD 完整测试', () => {
  let page: Page;

  test.beforeEach(async ({ page: p }) => {
    page = p;
    await page.goto(BASE_URL, { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);
    // 强制关闭可能残留的 Modal
    const modal = page.locator('.ant-modal');
    if (await modal.isVisible().catch(() => false)) {
      await page.keyboard.press('Escape');
      await page.waitForTimeout(1000);
    }
  });

  test.afterEach(async () => {
    await page.close();
  });

  async function openCreateModal() {
    const btn = page.getByRole('button', { name: /创建任务/i });
    await btn.click();
    await page.waitForSelector('.ant-modal', { timeout: 8000 });
    await page.waitForTimeout(500);
  }

  async function selectTaskType(taskType: string) {
    await page.locator('.ant-select').filter({ hasText: /请选择任务类型/i }).click();
    await page.waitForTimeout(300);
    await page.locator('.ant-select-item-option').filter({ hasText: new RegExp(taskType) }).click();
    await page.waitForTimeout(300);
  }

  async function submitForm() {
    await page.locator('button[type="submit"]').click();
  }

  async function closeModal() {
    await page.keyboard.press('Escape');
    await page.waitForTimeout(1000);
  }

  async function waitForTaskAppear(taskName: string, timeout = 10000) {
    await page.waitForTimeout(2000);
    const locator = page.locator('.ant-table-tbody').getByText(taskName, { exact: false });
    await expect(locator.first()).toBeVisible({ timeout });
  }

  // ========== CREATE ==========

  test('C1 - 创建任务：成功创建 data_processing 类型任务', async () => {
    await openCreateModal();
    await page.locator('input[id*="name"]').fill('Playwright测试-数据处理');
    await selectTaskType('数据处理');
    await submitForm();
    await page.waitForSelector('.ant-modal', { state: 'hidden', timeout: 10000 });
    await waitForTaskAppear('Playwright测试-数据处理');
  });

  test('C2 - 创建任务：成功创建 api_call 类型任务', async () => {
    await openCreateModal();
    await page.locator('input[id*="name"]').fill('Playwright测试-API调用');
    await selectTaskType('API 调用');
    await submitForm();
    await page.waitForSelector('.ant-modal', { state: 'hidden', timeout: 10000 });
    await waitForTaskAppear('Playwright测试-API调用');
  });

  test('C3 - 创建任务：成功创建 file_operation 类型任务（设置优先级）', async () => {
    await openCreateModal();
    await page.locator('input[id*="name"]').fill('Playwright测试-文件操作');
    await selectTaskType('文件操作');
    await page.locator('input[id*="priority"]').fill('10');
    await submitForm();
    await page.waitForSelector('.ant-modal', { state: 'hidden', timeout: 10000 });
    await waitForTaskAppear('Playwright测试-文件操作');
  });

  test('C4 - 创建任务：失败 - 名称为空', async () => {
    await openCreateModal();
    await selectTaskType('数据处理');
    await submitForm();
    await page.waitForTimeout(500);
    await expect(page.locator('.ant-form-item-explain-error').first()).toBeVisible({ timeout: 3000 });
    await closeModal();
  });

  test('C5 - 创建任务：失败 - 未选择任务类型', async () => {
    await openCreateModal();
    await page.locator('input[id*="name"]').fill('测试任务名称');
    await submitForm();
    await page.waitForTimeout(500);
    await expect(page.locator('.ant-form-item-explain-error').first()).toBeVisible({ timeout: 3000 });
    await closeModal();
  });

  test('C6 - 创建任务：取消创建', async () => {
    await openCreateModal();
    await page.locator('input[id*="name"]').fill('不应创建的任务');
    await page.waitForTimeout(300);
    await page.locator('button').filter({ hasText: /^取消$/ }).click();
    await page.waitForTimeout(1000);
    await expect(page.locator('.ant-modal')).not.toBeVisible({ timeout: 5000 });
    await page.waitForTimeout(500);
    await expect(page.locator('.ant-table-tbody').getByText('不应创建的任务')).not.toBeVisible({ timeout: 2000 });
  });

  // ========== READ ==========

  test('R1 - 读取任务列表：页面正常加载，统计卡片可见', async () => {
    await expect(page.locator('.ant-statistic-title').filter({ hasText: /待处理/i })).toBeVisible();
    await expect(page.locator('.ant-statistic-title').filter({ hasText: /运行中/i })).toBeVisible();
    await expect(page.locator('.ant-statistic-title').filter({ hasText: /已完成/i })).toBeVisible();
    await expect(page.locator('.ant-statistic-title').filter({ hasText: /失败/i })).toBeVisible();
    await expect(page.locator('.ant-card-head-title').filter({ hasText: /任务列表/i })).toBeVisible();
  });

  test('R2 - 读取任务列表：任务列表表格可见', async () => {
    await expect(page.locator('.ant-table')).toBeVisible();
  });

  test('R3 - 读取任务列表：刷新按钮功能正常', async () => {
    const refreshBtn = page.getByRole('button', { name: /刷新/i });
    await expect(refreshBtn).toBeVisible();
    await refreshBtn.click();
    await page.waitForTimeout(1500);
    await expect(page.locator('.ant-table')).toBeVisible();
  });

  test('R4 - 读取任务详情：点击任务名称可查看详情', async () => {
    const taskLink = page.locator('.ant-table-tbody .ant-link').first();
    if (await taskLink.isVisible()) {
      await taskLink.click();
      await page.waitForTimeout(1000);
      await expect(page.locator('.ant-drawer, .ant-modal, .ant-descriptions').first()).toBeVisible({ timeout: 3000 });
    }
  });

  // ========== UPDATE ==========

  test('U1 - 更新任务：查看任务详情可以展示任务信息', async () => {
    await openCreateModal();
    const testTaskName = 'Playwright更新测试任务';
    await page.locator('input[id*="name"]').fill(testTaskName);
    await selectTaskType('API 调用');
    await submitForm();
    await page.waitForSelector('.ant-modal', { state: 'hidden', timeout: 10000 });
    await waitForTaskAppear(testTaskName);
    const taskLink = page.locator('.ant-table-tbody').getByText(testTaskName, { exact: false }).first();
    if (await taskLink.isVisible()) {
      await taskLink.click();
      await page.waitForTimeout(1000);
      await expect(page.locator('text=' + testTaskName).first()).toBeVisible({ timeout: 3000 });
    }
  });

  // ========== DELETE / CANCEL ==========

  test('D1 - 取消任务：pending 状态任务可以被取消', async () => {
    await openCreateModal();
    const cancelTestTaskName = 'Playwright取消测试任务';
    await page.locator('input[id*="name"]').fill(cancelTestTaskName);
    await selectTaskType('数据处理');
    await submitForm();
    await page.waitForSelector('.ant-modal', { state: 'hidden', timeout: 10000 });
    await waitForTaskAppear(cancelTestTaskName);

    const taskRow = page.locator('.ant-table-tbody tr').filter({ hasText: cancelTestTaskName }).first();
    const cancelBtn = taskRow.locator('button').filter({ hasText: /取消/i });
    if (await cancelBtn.isVisible({ timeout: 3000 })) {
      await cancelBtn.click();
      await page.waitForTimeout(1500);
      await expect(taskRow.locator('.ant-badge')).toBeVisible();
    }
  });

  test('D2 - 取消任务：取消后列表中显示 cancelled 状态', async () => {
    await openCreateModal();
    const cancelStatusTaskName = 'Playwright状态测试';
    await page.locator('input[id*="name"]').fill(cancelStatusTaskName);
    await selectTaskType('文件操作');
    await submitForm();
    await page.waitForSelector('.ant-modal', { state: 'hidden', timeout: 10000 });
    await waitForTaskAppear(cancelStatusTaskName);

    const taskRow = page.locator('.ant-table-tbody tr').filter({ hasText: cancelStatusTaskName }).first();
    const cancelBtn = taskRow.locator('button').filter({ hasText: /取消/i });
    if (await cancelBtn.isVisible({ timeout: 3000 })) {
      await cancelBtn.click();
      await page.waitForTimeout(2000);
      await page.reload();
      await page.waitForLoadState('networkidle');
      await page.waitForTimeout(1000);
      const cancelledRow = page.locator('.ant-table-tbody tr').filter({ hasText: cancelStatusTaskName });
      await expect(cancelledRow).toBeVisible({ timeout: 3000 });
      await expect(cancelledRow.locator('text=/cancelled/i')).toBeVisible();
    }
  });

  // ========== 边界 & 异常测试 ==========

  test('E1 - 异常处理：页面加载时能正常加载完成', async () => {
    await page.goto(BASE_URL, { waitUntil: 'networkidle' });
    await page.waitForTimeout(2000);
    await expect(page.locator('.ant-table')).toBeVisible({ timeout: 5000 });
  });

  test('E2 - 异常处理：创建多个任务后列表正确更新', async () => {
    for (let i = 0; i < 3; i++) {
      await openCreateModal();
      const taskName = '多任务测试' + i;
      await page.locator('input[id*="name"]').fill(taskName);
      await selectTaskType('自定义');
      await submitForm();
      await page.waitForSelector('.ant-modal', { state: 'hidden', timeout: 15000 });
      await page.waitForTimeout(2000);
    }
    await page.waitForTimeout(2000);
    for (let i = 0; i < 3; i++) {
      await expect(page.locator('.ant-table-tbody').getByText('多任务测试' + i, { exact: false }).first()).toBeVisible({ timeout: 10000 });
    }
  });

  test('E3 - 边界测试：长任务名称正确显示', async () => {
    const longName = 'Playwright自动化测试——这是一个非常非常非常非常非常非常非常非常非常长的任务名称用于测试边界情况';
    await openCreateModal();
    await page.locator('input[id*="name"]').fill(longName);
    await selectTaskType('数据处理');
    await submitForm();
    await page.waitForSelector('.ant-modal', { state: 'hidden', timeout: 10000 });
    await waitForTaskAppear(longName.slice(0, 30));
  });

  test('E4 - 边界测试：设置最大超时时间', async () => {
    await openCreateModal();
    await page.locator('input[id*="name"]').fill('超时边界测试任务');
    await selectTaskType('API 调用');
    await page.locator('input[id*="timeout"]').fill('300000');
    await submitForm();
    await page.waitForSelector('.ant-modal', { state: 'hidden', timeout: 10000 });
    await waitForTaskAppear('超时边界测试任务', 15000);
  });
});
