import { test, expect } from '@playwright/test';

function uniqueName(prefix: string) {
  return `${prefix}-${Date.now()}-${Math.floor(Math.random() * 1000)}`;
}

test.describe('项目需求管理', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('http://localhost:3000/projects');
    // wait for project cards or requirements view to render
    await page.waitForLoadState('networkidle');
  });

  test('PR1 - 项目列表页面加载成功', async ({ page }) => {
    await expect(page.getByRole('button', { name: '新建项目' })).toBeVisible();
  });

  test('PR2 - 成功创建新项目', async ({ page }) => {
    const projectName = uniqueName('E2E项目');

    await page.getByRole('button', { name: '新建项目' }).click();
    await expect(page.getByRole('dialog', { name: '新建项目' })).toBeVisible();

    await page.getByLabel('项目名称').fill(projectName);
    await page.getByLabel('Git 仓库地址').fill('https://github.com/test/e2e-repo');
    await page.getByLabel('默认分支').fill('main');

    await page.locator('.ant-drawer-footer button[type="submit"], .ant-drawer-body button[type="submit"]').click();

    // Wait for drawer to close and toast success
    await expect(page.getByText('创建项目成功')).toBeVisible();
    await expect(page.getByRole('dialog', { name: '新建项目' })).not.toBeVisible();

    // Verify card appears in list
    await expect(page.locator('.ant-card').getByText(projectName)).toBeVisible();
  });

  test('PR3 - 进入项目查看需求列表', async ({ page }) => {
    // Create a project first
    const projectName = uniqueName('E2E进入项目');
    await page.getByRole('button', { name: '新建项目' }).click();
    await page.getByLabel('项目名称').fill(projectName);
    await page.getByLabel('Git 仓库地址').fill('https://github.com/test/e2e-repo');
    await page.getByLabel('默认分支').fill('main');
    await page.locator('.ant-drawer-footer button[type="submit"], .ant-drawer-body button[type="submit"]').click();
    await expect(page.getByText('创建项目成功')).toBeVisible();

    // Click project card
    await page.locator('.ant-card').getByText(projectName).click();

    // Should see requirements toolbar
    await expect(page.getByRole('button', { name: '新建需求' })).toBeVisible();
    await expect(page.locator('.ant-table')).toBeVisible();
  });

  test('PR4 - 成功创建新需求', async ({ page }) => {
    const projectName = uniqueName('E2E需求项目');
    const reqTitle = uniqueName('E2E需求标题');

    // Create project
    await page.getByRole('button', { name: '新建项目' }).click();
    await page.getByLabel('项目名称').fill(projectName);
    await page.getByLabel('Git 仓库地址').fill('https://github.com/test/e2e-repo');
    await page.getByLabel('默认分支').fill('main');
    await page.locator('.ant-drawer-footer button[type="submit"], .ant-drawer-body button[type="submit"]').click();
    await expect(page.getByText('创建项目成功')).toBeVisible();

    // Enter project
    await page.locator('.ant-card').getByText(projectName).click();
    await expect(page.getByRole('button', { name: '新建需求' })).toBeVisible();

    // Create requirement
    await page.getByRole('button', { name: '新建需求' }).click();
    await expect(page.getByRole('dialog', { name: '新建需求' })).toBeVisible();

    await page.getByLabel('需求标题').fill(reqTitle);
    await page.getByLabel('需求描述').fill('这是E2E测试自动创建的需求描述');
    await page.getByLabel('验收标准').fill('验收标准：E2E测试通过');

    // Fill workspace root if required
    const workspaceInput = page.getByLabel('临时工作目录根路径');
    if (await workspaceInput.isVisible().catch(() => false)) {
      await workspaceInput.fill('/tmp/e2e-workspace');
    }

    await page.locator('.ant-modal-footer button[type="submit"], .ant-modal-body button[type="submit"]').first().click();

    await expect(page.getByText('创建需求成功')).toBeVisible();
    await expect(page.getByRole('dialog', { name: '新建需求' })).not.toBeVisible();

    // Verify in table
    await expect(page.locator('.ant-table').getByText(reqTitle)).toBeVisible();
  });

  test('PR5 - 面包屑可返回项目列表', async ({ page }) => {
    const projectName = uniqueName('E2E面包屑项目');
    await page.getByRole('button', { name: '新建项目' }).click();
    await page.getByLabel('项目名称').fill(projectName);
    await page.getByLabel('Git 仓库地址').fill('https://github.com/test/e2e-repo');
    await page.getByLabel('默认分支').fill('main');
    await page.locator('.ant-drawer-footer button[type="submit"], .ant-drawer-body button[type="submit"]').click();
    await expect(page.getByText('创建项目成功')).toBeVisible();

    await page.locator('.ant-card').getByText(projectName).click();
    await expect(page.getByRole('button', { name: '新建需求' })).toBeVisible();

    // Click breadcrumb back
    await page.locator('.ant-breadcrumb').getByText('项目列表').click();
    await expect(page.getByRole('button', { name: '新建项目' })).toBeVisible();
  });

  test('PR6 - 编辑项目信息', async ({ page }) => {
    const projectName = uniqueName('E2E编辑项目');
    const newName = uniqueName('E2E编辑后');

    await page.getByRole('button', { name: '新建项目' }).click();
    await page.getByLabel('项目名称').fill(projectName);
    await page.getByLabel('Git 仓库地址').fill('https://github.com/test/e2e-repo');
    await page.getByLabel('默认分支').fill('main');
    await page.locator('.ant-drawer-footer button[type="submit"], .ant-drawer-body button[type="submit"]').click();
    await expect(page.getByText('创建项目成功')).toBeVisible();

    // Edit directly from project list card actions
    const card = page.locator('.ant-card').filter({ hasText: projectName });
    await card.locator('[data-icon="edit"]').click();

    await expect(page.getByRole('dialog', { name: '编辑项目' })).toBeVisible();
    await page.getByLabel('项目名称').fill(newName);
    await page.locator('.ant-drawer-footer button[type="submit"], .ant-drawer-body button[type="submit"]').click();
    await expect(page.getByText('更新项目成功')).toBeVisible();
  });

  test('PR7 - 删除项目', async ({ page }) => {
    const projectName = uniqueName('E2E删除项目');

    await page.getByRole('button', { name: '新建项目' }).click();
    await page.getByLabel('项目名称').fill(projectName);
    await page.getByLabel('Git 仓库地址').fill('https://github.com/test/e2e-repo');
    await page.getByLabel('默认分支').fill('main');
    await page.locator('.ant-drawer-footer button[type="submit"], .ant-drawer-body button[type="submit"]').click();
    await expect(page.getByText('创建项目成功')).toBeVisible();

    // Delete directly from project list card actions
    const card = page.locator('.ant-card').filter({ hasText: projectName });
    await card.locator('[data-icon="delete"]').click();

    // Confirm popconfirm
    await page.getByRole('button', { name: /确\s*定/ }).click();
    await expect(page.getByText('删除项目成功')).toBeVisible();
    await expect(card).not.toBeVisible();
  });
});
