import {expect, test} from '@playwright/test';

test.describe('Integration: Navigation with real backend', () => {
    test('app loads and sidebar is visible', async ({page}) => {
        await page.goto('/');
        const nav = page.locator('nav');
        await expect(nav).toBeVisible({timeout: 10000});
    });

    test('menu data loads from real API', async ({page}) => {
        await page.goto('/');

        const navButtons = page.locator('nav button');
        await expect(navButtons.first()).toBeVisible({timeout: 10000});
    });

    test('navigating to /profile works', async ({page}) => {
        await page.goto('/profile');

        await expect(
            page.getByRole('main').getByText('Профиль')
        ).toBeVisible();
    });

    test('navigating to /reports works', async ({page}) => {
        await page.goto('/reports');
        await expect(page.getByText('отчёты')).toBeVisible({timeout: 10000});
    });

    test('root redirects to /profile', async ({page}) => {
        await page.goto('/');
        await page.waitForURL('**/profile', {timeout: 5000});
    });
});
