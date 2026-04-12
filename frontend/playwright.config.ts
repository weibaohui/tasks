import { defineConfig, devices } from '@playwright/test';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: 'list',
  use: {
    baseURL: 'http://localhost:3000',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'setup',
      testMatch: /auth\.setup\.ts/,
    },
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        storageState: path.join(__dirname, 'playwright/.auth/user.json'),
      },
      dependencies: ['setup'],
      testIgnore: /login\.spec\.ts/,
    },
    {
      name: 'chromium-no-auth',
      use: { ...devices['Desktop Chrome'] },
      testMatch: /login\.spec\.ts/,
    },
  ],
  webServer: [
    {
      command: 'cd ../backend && ./bin/taskmanager-server create-admin 2>/dev/null || true && ./bin/taskmanager-server',
      url: 'http://localhost:13618',
      timeout: 30 * 1000,
      reuseExistingServer: true,
    },
    {
      command: 'pnpm exec vite preview --port 3000',
      url: 'http://localhost:3000',
      timeout: 30 * 1000,
      reuseExistingServer: true,
    },
  ],
});
