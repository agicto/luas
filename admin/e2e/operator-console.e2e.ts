import { expect, test, type Page } from '@playwright/test';

// Accounts created by the API seeders that `make dev` runs; `make dev` grants admin@ the operator role.
const operator = { identifier: 'admin@example.com', password: 'secret' };
const member = { identifier: 'user@example.com', password: 'secret' };

async function signIn(page: Page, account: { identifier: string; password: string }) {
  await page.goto('/login');
  await page.getByLabel('Username or email').fill(account.identifier);
  await page.getByLabel('Password').fill(account.password);
  await page.getByRole('button', { name: 'Sign in' }).click();
}

test('an operator signs in, sees live API state, and signs out', async ({ page }) => {
  await signIn(page, operator);
  await expect(page).toHaveURL(/\/console/);
  // System status comes from GET /v1/operator/system, so this proves the API and database are live.
  await expect(page.getByRole('term').filter({ hasText: 'Database' })).toBeVisible();
  await expect(page.getByRole('definition').filter({ hasText: /^Available$/ })).toBeVisible();

  await page.getByRole('link', { name: 'Users' }).click();
  await expect(page.getByText('user · user@example.com')).toBeVisible();

  await page.getByRole('link', { name: 'Audit log' }).click();
  await expect(page.getByRole('cell', { name: /operator\.session/ }).first()).toBeVisible();

  await page.getByRole('button', { name: 'Sign out' }).click();
  await expect(page).toHaveURL(/\/login/);
  await page.goto('/console/users');
  await expect(page).toHaveURL(/\/login/);
});

test('a valid account without an operator grant cannot sign in', async ({ page }) => {
  await signIn(page, member);
  await expect(page.getByRole('alert')).toHaveText(
    'This account does not have platform-operator access.',
  );
  await expect(page).toHaveURL(/\/login/);
});

test('the organization directory loads from the API', async ({ page }) => {
  await signIn(page, operator);
  await page.getByRole('link', { name: 'Organizations' }).click();
  await expect(page.getByRole('heading', { level: 1, name: 'Organizations' })).toBeVisible();
  await expect(page.getByRole('alert')).toHaveCount(0);
});
