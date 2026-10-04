import { expect, test, type Page } from '@playwright/test';

// The regular account created by the API seeders that `make dev` runs.
const member = { email: 'user@example.com', password: 'secret', nickname: 'Regular User' };

async function signIn(page: Page, password: string) {
  await page.goto('/login');
  await page.getByLabel('Email').fill(member.email);
  await page.getByLabel('Password').fill(password);
  await page.getByRole('button', { name: 'Sign In' }).click();
}

test('a member signs in through the API adapter, sees live account data, and signs out', async ({
  page,
}) => {
  await signIn(page, member.password);
  await expect(page).toHaveURL(/\/console$/);

  // The nickname comes from the Go API through the server-owned adapter, not from mock state.
  await page.getByRole('banner').getByRole('button', { name: 'Profile', exact: true }).click();
  await expect(page.getByText(member.nickname)).toBeVisible();

  await page.getByRole('menuitem').filter({ hasText: 'Sign Out' }).click();
  await expect(page).toHaveURL(/\/login/);
  await page.goto('/console');
  await expect(page).toHaveURL(/\/login\?returnUrl=%2Fconsole/);
});

test('a wrong password keeps the visitor on the sign-in page', async ({ page }) => {
  await signIn(page, 'not-the-password');
  // Next.js also renders an empty route-announcer alert, so match the form alert by its text.
  await expect(
    page.getByRole('alert').filter({ hasText: 'Invalid email or password' })
  ).toBeVisible();
  await expect(page).toHaveURL(/\/login/);
});
