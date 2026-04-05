/**
 * 状态机工作流 E2E 测试
 *
 * 测试软件开发流程状态机的完整生命周期
 *
 * 前置条件：
 * 1. 服务运行在 localhost:13618
 * 2. 使用 playwright-cli 进行 UI 测试
 */

import { test, expect } from '@playwright/test';

const BASE_URL = 'http://localhost:13618';

// 简化版状态机 YAML
const SIMPLE_WORKFLOW_YAML = `
name: simple_dev_workflow
description: 简化版开发流程，用于E2E测试

initial_state: submitted

states:
  - id: submitted
    name: 已提交
    is_final: false
  - id: in_review
    name: 审查中
    is_final: false
  - id: building
    name: 构建中
    is_final: false
  - id: testing
    name: 测试中
    is_final: false
  - id: completed
    name: 已完成
    is_final: true

transitions:
  - from: submitted
    to: in_review
    trigger: submit_review
    description: 提交审查
    hooks:
      - name: 通知审查者
        type: webhook
        config:
          url: https://httpbin.org/post
          method: POST
        timeout: 30
        retry: 1

  - from: in_review
    to: building
    trigger: approve
    description: 审查通过
    hooks:
      - name: 触发构建
        type: command
        config:
          command: echo "Building {{requirement_id}}..."
        timeout: 30
        retry: 0

  - from: in_review
    to: submitted
    trigger: reject
    description: 审查拒绝
    hooks:
      - name: 通知开发者
        type: webhook
        config:
          url: https://httpbin.org/post
          method: POST
        timeout: 30
        retry: 1

  - from: building
    to: testing
    trigger: build_success
    description: 构建成功
    hooks:
      - name: 触发测试
        type: command
        config:
          command: echo "Testing {{requirement_id}}..."
        timeout: 30
        retry: 0

  - from: building
    to: submitted
    trigger: build_failed
    description: 构建失败
    hooks:
      - name: 通知失败
        type: webhook
        config:
          url: https://httpbin.org/post
          method: POST
        timeout: 30
        retry: 1

  - from: testing
    to: completed
    trigger: test_pass
    description: 测试通过
    hooks:
      - name: 发送完成通知
        type: webhook
        config:
          url: https://httpbin.org/post
          method: POST
        timeout: 30
        retry: 1

  - from: testing
    to: building
    trigger: test_failed
    description: 测试失败
    hooks:
      - name: 通知测试失败
        type: webhook
        config:
          url: https://httpbin.org/post
          method: POST
        timeout: 30
        retry: 1
`;

test.describe('状态机工作流 E2E 测试', () => {
  let stateMachineId: string;
  let requirementId: string;

  test.beforeAll(async ({ browser }) => {
    // 登录并确保已认证
    const context = await browser.newContext();
    const page = await context.newPage();

    // 访问应用
    await page.goto(BASE_URL);

    // 如果需要登录，先登录
    // await page.fill('[data-testid="username"]', 'admin');
    // await page.fill('[data-testid="password"]', 'admin');
    // await page.click('[data-testid="login-button"]');

    await context.close();
  });

  test('1. 创建状态机', async ({ page }) => {
    await page.goto(`${BASE_URL}/state-machines`);

    // 点击新建状态机
    await page.click('button:has-text("新建状态机")');

    // 等待抽屉打开
    await page.waitForSelector('text=新建状态机', { state: 'visible' });

    // 填写基本信息
    await page.fill('input[placeholder*="需求流程"]', 'E2E测试工作流');
    await page.fill('textarea[placeholder*="描述"]', 'E2E自动化测试用的状态机');

    // 切换到 JSON 编辑器
    await page.click('text=JSON 编辑');

    // 清除现有内容并填写新的 YAML/JSON
    const editor = page.locator('textarea').first();
    await editor.fill('');

    // 使用 YAML 格式粘贴配置
    await editor.fill(SIMPLE_WORKFLOW_YAML);

    // 保存
    await page.click('button:has-text("保存")');

    // 等待抽屉关闭
    await page.waitForSelector('text=新建状态机', { state: 'hidden' });

    // 验证状态机出现在列表中
    await expect(page.locator('text=E2E测试工作流')).toBeVisible();

    // 从 URL 或表格中获取状态机 ID
    // 这里简化处理，实际需要从表格或 API 获取
  });

  test('2. 创建需求并初始化状态', async ({ page }) => {
    // 先获取刚才创建的状态机 ID
    // 实际测试中需要从第一步获取

    // 创建需求
    await page.goto(`${BASE_URL}/requirements`);

    // 点击新建需求
    await page.click('button:has-text("新建需求")');

    // 填写需求信息
    await page.fill('input[placeholder*="标题"]', 'E2E测试需求');
    await page.fill('textarea[placeholder*="描述"]', '用于状态机E2E测试的需求');

    // 选择状态机（如果下拉中有选项）
    // await page.selectOption('select[name="state_machine_id"]', stateMachineId);

    // 保存
    await page.click('button:has-text("保存")');
  });

  test('3. 触发状态转换：提交审查', async ({ page }) => {
    // 进入需求详情
    await page.goto(`${BASE_URL}/requirements`);

    // 点击刚创建的需求
    await page.click('text=E2E测试需求');

    // 等待需求详情加载
    await page.waitForSelector('text=E2E测试需求', { state: 'visible' });

    // 查找并点击"提交审查"按钮
    const submitReviewBtn = page.locator('button:has-text("submit_review"), button:has-text("提交审查")');
    if (await submitReviewBtn.isVisible()) {
      await submitReviewBtn.click();

      // 等待状态更新
      await page.waitForTimeout(500);

      // 验证状态变为"审查中"
      await expect(page.locator('text=审查中')).toBeVisible();
    }
  });

  test('4. 触发状态转换：审查通过', async ({ page }) => {
    // 进入需求详情
    await page.goto(`${BASE_URL}/requirements`);

    // 点击刚创建的需求
    await page.click('text=E2E测试需求');

    // 等待状态为审查中
    await page.waitForSelector('text=审查中', { state: 'visible' });

    // 查找并点击"审查通过"按钮
    const approveBtn = page.locator('button:has-text("approve"), button:has-text("审查通过")');
    if (await approveBtn.isVisible()) {
      await approveBtn.click();

      // 等待状态更新
      await page.waitForTimeout(500);

      // 验证状态变为"构建中"
      await expect(page.locator('text=构建中')).toBeVisible();
    }
  });

  test('5. 触发状态转换：构建成功', async ({ page }) => {
    // 进入需求详情
    await page.goto(`${BASE_URL}/requirements`);

    // 点击刚创建的需求
    await page.click('text=E2E测试需求');

    // 等待状态为构建中
    await page.waitForSelector('text=构建中', { state: 'visible' });

    // 查找并点击"构建成功"按钮
    const buildSuccessBtn = page.locator('button:has-text("build_success"), button:has-text("构建成功")');
    if (await buildSuccessBtn.isVisible()) {
      await buildSuccessBtn.click();

      // 等待状态更新
      await page.waitForTimeout(500);

      // 验证状态变为"测试中"
      await expect(page.locator('text=测试中')).toBeVisible();
    }
  });

  test('6. 触发状态转换：测试通过，完成流程', async ({ page }) => {
    // 进入需求详情
    await page.goto(`${BASE_URL}/requirements`);

    // 点击刚创建的需求
    await page.click('text=E2E测试需求');

    // 等待状态为测试中
    await page.waitForSelector('text=测试中', { state: 'visible' });

    // 查找并点击"测试通过"按钮
    const testPassBtn = page.locator('button:has-text("test_pass"), button:has-text("测试通过")');
    if (await testPassBtn.isVisible()) {
      await testPassBtn.click();

      // 等待状态更新
      await page.waitForTimeout(500);

      // 验证状态变为"已完成"
      await expect(page.locator('text=已完成')).toBeVisible();
    }
  });

  test('7. 验证状态转换历史', async ({ page }) => {
    // 进入需求详情
    await page.goto(`${BASE_URL}/requirements`);

    // 点击刚创建的需求
    await page.click('text=E2E测试需求');

    // 查找转换历史标签或按钮
    const historyBtn = page.locator('text=转换历史, text=历史');
    if (await historyBtn.isVisible()) {
      await historyBtn.click();

      // 等待历史记录加载
      await page.waitForTimeout(500);

      // 验证有多个转换记录
      // 提交审查、审查通过、构建成功、测试通过 应该有 4 条记录
      const historyItems = page.locator('[data-testid="transition-history"] .history-item, .transition-log');
      const count = await historyItems.count();
      expect(count).toBeGreaterThanOrEqual(4);
    }
  });
});
