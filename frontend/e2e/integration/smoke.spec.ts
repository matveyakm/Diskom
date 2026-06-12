import {expect, test} from '@playwright/test';

test.describe('Integration: Smoke tests', () => {
    test('console has no uncaught errors on main pages', async ({page}) => {
        const errors: string[] = [];
        page.on('pageerror', err => errors.push(err.message));

        await page.goto('/profile');
        await page.waitForLoadState('networkidle');

        expect(errors).toHaveLength(0);
    });

    test('table page loads without critical errors', async ({page}) => {
        const errors: string[] = [];
        page.on('pageerror', err => errors.push(err.message));

        await page.goto('/');
        await page.waitForLoadState('networkidle');

        const navButtons = page.locator('nav button');
        const count = await navButtons.count();

        for (let i = 0; i < Math.min(count, 3); i++) {
            const href = await navButtons.nth(i).locator('..').getAttribute('href');
            if (href && href !== '#' && !href.startsWith('/table/')) {
                await page.goto(href);
                await page.waitForLoadState('networkidle');
                expect(errors).toHaveLength(0);
                return;
            }
        }
    });
});
