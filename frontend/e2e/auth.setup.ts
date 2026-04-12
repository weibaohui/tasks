import { test as setup, expect } from '@playwright/test';
import { existsSync, mkdirSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const authFile = path.join(__dirname, '../playwright/.auth/user.json');

setup('authenticate', async ({ page }) => {
  await page.goto('http://localhost:3000/login');

  await page.getByPlaceholder('请输入用户名').fill('admin');
  await page.getByPlaceholder('请输入密码').fill('admin123');
  await page.getByRole('button', { name: /登\s*录/ }).click();

  // Wait for navigation to dashboard
  await expect(page).toHaveURL(/\/dashboard/, { timeout: 10000 });
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();

  // Ensure directory exists
  const dir = path.dirname(authFile);
  if (!existsSync(dir)) {
    mkdirSync(dir, { recursive: true });
  }

  await page.context().storageState({ path: authFile });
});
