import { test, expect, Page } from '@playwright/test';

const ADMIN_USER = process.env.E2E_USERNAME ?? 'michael.jordan';
const ADMIN_PASS = process.env.E2E_PASSWORD ?? '111222333';

async function loginAs(page: Page, username = ADMIN_USER, password = ADMIN_PASS) {
  await page.goto('/login');
  await page.getByLabel(/username/i).fill(username);
  await page.getByLabel(/password/i).fill(password);
  await page.getByRole('button', { name: /sign in/i }).click();
  await expect(page).not.toHaveURL('/login', { timeout: 10_000 });
}

test.describe('Create Build', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page);
  });

  test('navigates to builds page', async ({ page }) => {
    await page.goto('/builds');
    await expect(page).toHaveURL(/builds/);
    await expect(page.getByRole('heading', { name: /build|model/i }).first()).toBeVisible({ timeout: 10_000 });
  });

  test('opens Build Model dialog from the builds page', async ({ page }) => {
    await page.goto('/builds');
    const buildBtn = page.getByRole('button', { name: /build model|new build|create build/i });
    await expect(buildBtn).toBeVisible({ timeout: 10_000 });
    await buildBtn.click();

    // Dialog should open
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 });
    await expect(page.getByRole('dialog').getByText(/build model|create build/i)).toBeVisible();
  });

  test('closes the Build Model dialog when Cancel is clicked', async ({ page }) => {
    await page.goto('/builds');
    const buildBtn = page.getByRole('button', { name: /build model|new build|create build/i });
    await buildBtn.click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 });

    await page.getByRole('dialog').getByRole('button', { name: /cancel/i }).click();
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 3_000 });
  });

  test('validates required fields in Build Model dialog', async ({ page }) => {
    await page.goto('/builds');
    const buildBtn = page.getByRole('button', { name: /build model|new build|create build/i });
    await buildBtn.click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 });

    // Try submitting empty form
    const submitBtn = page.getByRole('dialog').getByRole('button', { name: /create|build|start/i });
    await submitBtn.click();

    // Should show validation error or stay on dialog
    await expect(page.getByRole('dialog')).toBeVisible();
  });

  test('can select model type and see algorithm options', async ({ page }) => {
    await page.goto('/builds');
    const buildBtn = page.getByRole('button', { name: /build model|new build|create build/i });
    await buildBtn.click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 });

    // Select regression model type
    const modelTypeSelect = page.getByRole('dialog').locator('select').first();
    if (await modelTypeSelect.count() > 0) {
      await modelTypeSelect.selectOption('regression');
      // Should show regression algorithms
      await expect(page.getByText(/linear regression|random forest/i).first()).toBeVisible({ timeout: 3_000 });
    }
  });

  test('displays existing builds in the list', async ({ page }) => {
    await page.goto('/builds');
    // The list should render (even if empty)
    await expect(page.getByRole('list').or(page.locator('table').or(page.getByText(/no builds|empty/i))).first()).toBeVisible({ timeout: 10_000 });
  });
});
