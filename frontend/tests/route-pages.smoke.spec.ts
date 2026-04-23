import { expect, test, type Locator, type Page } from '@playwright/test';

const adminUsername = process.env.CONSOLE_E2E_ADMIN_USERNAME || 'admin';
const adminPassword = process.env.CONSOLE_E2E_ADMIN_PASSWORD || 'admin';

async function clickPrimarySubmit(page: Page) {
  await page.locator('button.ant-btn-primary').last().click();
}

async function ensureSignedIn(page: Page) {
  await page.goto('/');

  if (page.url().includes('/init')) {
    await page.getByPlaceholder(/用户名|Username/).fill(adminUsername);
    await page.getByPlaceholder(/密码|Password/).fill(adminPassword);
    await page.getByPlaceholder(/确认密码|Confirm Password/).fill(adminPassword);
    await clickPrimarySubmit(page);
    await page.waitForURL(/\/login(?:\?|$)/);
  }

  if (page.url().includes('/login')) {
    await page.getByPlaceholder(/用户名|Username/).fill(adminUsername);
    await page.getByPlaceholder(/密码|Password/).fill(adminPassword);
    await clickPrimarySubmit(page);
  }

  await page.waitForURL(/\/dashboard(?:\?|$)|\/route(?:\?|$)|\/ai\/route(?:\?|$)/);
}

async function expectDrawerVisible(drawer: Locator) {
  await expect(drawer).toBeVisible();
  await expect(drawer.locator('.ant-drawer-body')).toBeVisible();
}

async function closeDrawer(drawer: Locator) {
  await drawer.getByRole('button', { name: /取\s*消|Cancel/ }).click();
  await expect(drawer).toBeHidden();
}

test.describe('console route page smoke', () => {
  test.beforeEach(async ({ page }) => {
    await ensureSignedIn(page);
  });

  test('opens gateway route drawer with auth controls', async ({ page }) => {
    await page.goto('/route');
    await expect(page.getByText('路由配置')).toBeVisible();

    await page.getByRole('button', { name: '新增路由' }).click();
    const drawer = page.locator('.ant-drawer-content').last();
    await expectDrawerVisible(drawer);
    await expect(drawer.locator('.ant-drawer-title')).toHaveText('新增路由');
    await expect(drawer.getByText('目标服务', { exact: true })).toBeVisible();
    await expect(drawer.getByText('请求认证', { exact: true })).toBeVisible();
    await drawer.getByRole('switch').click();
    await expect(drawer.getByText('允许访问的部门', { exact: true })).toBeVisible();
    await expect(drawer.getByText('允许访问的用户等级', { exact: true })).toBeVisible();
    await closeDrawer(drawer);
  });

  test('opens AI route drawer with provider-bound target model section and required auth', async ({ page }) => {
    await page.goto('/ai/route');
    await expect(page.getByText('AI 路由管理')).toBeVisible();

    await page.getByRole('button', { name: '创建 AI 路由' }).click();
    const drawer = page.locator('.ant-drawer-content').last();
    await expectDrawerVisible(drawer);
    await expect(drawer.locator('.ant-drawer-title')).toHaveText('创建 AI 路由');
    await expect(drawer.getByText('目标 AI 服务', { exact: true })).toBeVisible();
    await expect(drawer.getByRole('button', { name: '新增目标 AI 服务' })).toBeVisible();
    await expect(drawer.getByText('请求认证', { exact: true })).toBeVisible();
    await expect(drawer.getByText('允许访问的部门', { exact: true })).toBeVisible();
    await expect(drawer.getByText('允许访问的用户等级', { exact: true })).toBeVisible();
    await expect(drawer.getByText('启用请求认证', { exact: true })).toBeVisible();
    await closeDrawer(drawer);
  });
});
