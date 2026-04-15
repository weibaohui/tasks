# 需求列表展示耗时 - 测试说明

## 本地验证

- 安装依赖（前端目录）：
  - `pnpm install`
- TypeScript 构建：
  - `pnpm run build`
- ESLint：
  - `pnpm run lint`

## E2E 验证

- 启动后端与前端（按项目既有流程）
- 执行 Playwright：
  - `pnpm exec playwright test e2e/projectRequirement.spec.ts`
- 校验点：
  - “项目需求管理”页面表格中出现“耗时”列头

