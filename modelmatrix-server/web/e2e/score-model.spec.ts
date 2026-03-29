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

test.describe('Score Model', () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page);
  });

  test('navigates to models page', async ({ page }) => {
    await page.goto('/models');
    await expect(page).toHaveURL(/models/);
    await expect(page.getByRole('heading', { name: /model/i }).first()).toBeVisible({ timeout: 10_000 });
  });

  test('shows Score button for active models', async ({ page }) => {
    await page.goto('/models');

    // If there are any active models, they should have a Score button
    const activeModels = page.getByTestId('model-card').filter({ hasText: /active/i });
    const count = await activeModels.count();

    if (count > 0) {
      const scoreBtn = activeModels.first().getByRole('button', { name: /score/i });
      await expect(scoreBtn).toBeVisible({ timeout: 5_000 });
    } else {
      // No active models — just verify the page loaded
      await expect(page.getByRole('heading', { name: /model/i }).first()).toBeVisible();
    }
  });

  test('opens Score dialog when Score button is clicked', async ({ page }) => {
    await page.goto('/models');

    const scoreBtn = page.getByRole('button', { name: /score/i }).first();
    if (await scoreBtn.count() === 0) {
      test.skip();
      return;
    }

    await scoreBtn.click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 });
    await expect(page.getByRole('dialog').getByText(/score|predict/i)).toBeVisible();
  });

  test('closes Score dialog on cancel', async ({ page }) => {
    await page.goto('/models');

    const scoreBtn = page.getByRole('button', { name: /score/i }).first();
    if (await scoreBtn.count() === 0) {
      test.skip();
      return;
    }

    await scoreBtn.click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 });

    await page.getByRole('dialog').getByRole('button', { name: /cancel/i }).click();
    await expect(page.getByRole('dialog')).not.toBeVisible({ timeout: 3_000 });
  });

  test('opens Retrain dialog', async ({ page }) => {
    await page.goto('/models');

    const retrainBtn = page.getByRole('button', { name: /retrain/i }).first();
    if (await retrainBtn.count() === 0) {
      test.skip();
      return;
    }

    await retrainBtn.click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5_000 });
    await expect(page.getByRole('dialog').getByText(/retrain/i)).toBeVisible();
  });

  test('model detail page shows model information', async ({ page }) => {
    await page.goto('/models');

    // Click on any model card or link
    const modelLinks = page.getByRole('link').filter({ hasText: /model/i });
    const modelCards = page.locator('[data-testid="model-card"]');

    if (await modelLinks.count() > 0) {
      await modelLinks.first().click();
    } else if (await modelCards.count() > 0) {
      await modelCards.first().click();
    } else {
      test.skip();
      return;
    }

    // Should navigate to a model detail page
    await expect(page).toHaveURL(/models\/[a-z0-9-]+/, { timeout: 5_000 });
    await expect(page.getByText(/algorithm|model type|status/i).first()).toBeVisible({ timeout: 10_000 });
  });
});
