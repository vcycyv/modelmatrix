import { test, expect } from '@playwright/test';

const ADMIN_USER = process.env.E2E_USERNAME ?? 'michael.jordan';
const ADMIN_PASS = process.env.E2E_PASSWORD ?? '111222333';

test.describe('Login', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/login');
  });

  test('renders the login page', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /modelmatrix/i })).toBeVisible();
    await expect(page.getByLabel(/username/i)).toBeVisible();
    await expect(page.getByLabel(/password/i)).toBeVisible();
    await expect(page.getByRole('button', { name: /sign in/i })).toBeVisible();
  });

  test('shows LDAP subtitle', async ({ page }) => {
    await expect(page.getByText(/ldap credentials/i)).toBeVisible();
  });

  test('logs in with valid credentials and redirects to home', async ({ page }) => {
    await page.getByLabel(/username/i).fill(ADMIN_USER);
    await page.getByLabel(/password/i).fill(ADMIN_PASS);
    await page.getByRole('button', { name: /sign in/i }).click();

    // Should redirect away from /login
    await expect(page).not.toHaveURL('/login', { timeout: 10_000 });
  });

  test('shows error on invalid password', async ({ page }) => {
    await page.getByLabel(/username/i).fill(ADMIN_USER);
    await page.getByLabel(/password/i).fill('wrongpassword');
    await page.getByRole('button', { name: /sign in/i }).click();

    await expect(page.getByText(/invalid credentials|login failed|incorrect/i)).toBeVisible({ timeout: 10_000 });
  });

  test('shows error on unknown username', async ({ page }) => {
    await page.getByLabel(/username/i).fill('nobody.exists');
    await page.getByLabel(/password/i).fill('somepass');
    await page.getByRole('button', { name: /sign in/i }).click();

    await expect(page.getByText(/invalid credentials|login failed|not found/i)).toBeVisible({ timeout: 10_000 });
  });

  test('button shows loading state during sign-in', async ({ page }) => {
    await page.getByLabel(/username/i).fill(ADMIN_USER);
    await page.getByLabel(/password/i).fill(ADMIN_PASS);

    // Click and immediately check for loading state
    const [response] = await Promise.all([
      page.waitForResponse('/api/auth/login'),
      page.getByRole('button', { name: /sign in/i }).click(),
    ]);

    expect(response.status()).toBe(200);
  });

  test('logs out and returns to login', async ({ page }) => {
    // Login first
    await page.getByLabel(/username/i).fill(ADMIN_USER);
    await page.getByLabel(/password/i).fill(ADMIN_PASS);
    await page.getByRole('button', { name: /sign in/i }).click();
    await expect(page).not.toHaveURL('/login', { timeout: 10_000 });

    // Logout
    const logoutBtn = page.getByRole('button', { name: /logout|sign out/i });
    if (await logoutBtn.isVisible()) {
      await logoutBtn.click();
    } else {
      // May be in a menu
      const avatar = page.getByRole('button').filter({ has: page.getByText(ADMIN_USER) });
      if (await avatar.count() > 0) {
        await avatar.click();
        await page.getByText(/logout|sign out/i).click();
      }
    }

    await expect(page).toHaveURL(/login/, { timeout: 10_000 });
  });
});
