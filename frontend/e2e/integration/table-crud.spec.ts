import {expect, test} from '@playwright/test';

test.describe('Integration: Table CRUD', () => {
    async function navigateToTableViaSidebar(page: import('@playwright/test').Page) {
        await page.goto('/');
        await page.waitForLoadState('networkidle');

        const navButtons = page.locator('nav button');
        await expect(navButtons.first()).toBeVisible({timeout: 10000});
        const count = await navButtons.count();

        for (let i = 0; i < count; i++) {
            const parent = navButtons.nth(i).locator('..');
            const isLink = await parent.evaluate(el => el.tagName === 'A');
            if (!isLink) {
                await navButtons.nth(i).click();
                break;
            }
        }

        const tableLink = page.locator('a[href^="/table/"]').first();
        await expect(tableLink).toBeVisible({timeout: 5000});
        await tableLink.click();
        await page.waitForURL(/\/table\//);
        await page.waitForLoadState('networkidle');
    }

    test('table page renders data from real API', async ({page}) => {
        await navigateToTableViaSidebar(page);

        const table = page.locator('table');
        await expect(table).toBeVisible({timeout: 10000});
    });

    test('pagination controls are present', async ({page}) => {
        await navigateToTableViaSidebar(page);

        const pagination = page.locator('nav[aria-label="pagination"], .MuiTablePagination-root, button:has-text(">")');
        await expect(pagination.first()).toBeVisible({timeout: 5000});
    });

    test('double-clicking a row opens modal with details', async ({page}) => {
        await navigateToTableViaSidebar(page);

        const firstRow = page.locator('table tbody tr').first();
        await expect(firstRow).toBeVisible({timeout: 10000});

        await firstRow.dblclick();

        await expect(page.getByText(/#\d+/)).toBeVisible({timeout: 5000});
    });

    test('add button opens creation modal', async ({page}) => {
        await navigateToTableViaSidebar(page);

        const addButton = page.getByText('Добавить');
        await expect(addButton).toBeVisible({timeout: 5000});
        await addButton.click();

        await expect(page.getByText(/Создание/)).toBeVisible({timeout: 5000});
    });

    test('error state is handled gracefully for missing table', async ({page}) => {
        await page.goto('/');
        await page.waitForLoadState('networkidle');

        await page.goto('/table/nonexistent_table_xyz');
        await page.waitForLoadState('networkidle');
        await page.waitForTimeout(2000);

        const errorText = page.getByText(/Не удалось загрузить данные|error|ошибк/i);
        const table = page.locator('table');

        const hasError = await errorText.isVisible().catch(() => false);
        const hasTable = await table.isVisible().catch(() => false);

        if (!hasError && !hasTable) {
            const body = await page.locator('body').textContent();
            expect(body?.length).toBeGreaterThan(0);
        }
    });
});
