import { test, expect } from '@playwright/test';

test.describe('登录页', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('http://localhost:3000/login');
  });

  test('L1 - 页面元素正确渲染', async ({ page }) => {
    await expect(page.getByText('任务平台登录')).toBeVisible();
    await expect(page.getByPlaceholder('请输入用户名')).toBeVisible();
    await expect(page.getByPlaceholder('请输入密码')).toBeVisible();
    await expect(page.getByRole('button', { name: /登\s*录/ })).toBeVisible();
  });

  test('L2 - 空表单提交显示验证错误', async ({ page }) => {
    await page.getByRole('button', { name: /登\s*录/ }).click();
    await expect(page.getByText('请输入用户名')).toBeVisible();
    await expect(page.getByText('请输入密码')).toBeVisible();
  });

  test('L3 - 错误凭据显示登录失败', async ({ page }) => {
    await page.getByPlaceholder('请输入用户名').fill('wronguser');
    await page.getByPlaceholder('请输入密码').fill('wrongpass');
    await page.getByRole('button', { name: /登\s*录/ }).click();
    await expect(page.getByText('登录失败，请检查用户名和密码')).toBeVisible();
  });

  test('L4 - 正确凭据登录后跳转到 Dashboard', async ({ page }) => {
    await page.getByPlaceholder('请输入用户名').fill('admin');
    await page.getByPlaceholder('请输入密码').fill('admin123');
    await page.getByRole('button', { name: /登\s*录/ }).click();

    await expect(page).toHaveURL(/\/dashboard/, { timeout: 10000 });
    await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
  });
});
