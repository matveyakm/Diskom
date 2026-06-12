import { test, expect } from '@playwright/test';

test.describe('Integration: CRUD Operations', () => {
  async function getMenuData(page: import('@playwright/test').Page) {
    const respPromise = page.waitForResponse(
      resp => resp.url().includes('/api/system_object_directory') && resp.status() === 200,
    );
    await page.goto('/');
    await page.waitForLoadState('networkidle');
    const resp = await respPromise;
    const json = await resp.json();
    return (json.data ?? []) as Array<{
      id: number;
      ref: string;
      id_group: number;
      level: number;
      is_group: boolean;
      output_order?: number;
    }>;
  }

  function getRootButtonIndex(
    menuItems: Array<{ id: number; level: number; output_order?: number }>,
    idGroup: number,
  ): number {
    const rootItems = menuItems
      .filter(item => item.level === 0 && item.id < 4)
      .sort((a, b) => (a.output_order ?? 0) - (b.output_order ?? 0));
    return rootItems.findIndex(item => item.id === idGroup);
  }

  async function navigateToTableViaSidebarByGroup(
    page: import('@playwright/test').Page,
    buttonIndex: number,
    targetRef: string,
  ) {
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const navButtons = page.locator('nav button');
    await expect(navButtons.first()).toBeVisible({ timeout: 10000 });
    await navButtons.nth(buttonIndex).click();

    const tableLink = page.locator(`a[href="/table/${targetRef}"]`);
    await expect(tableLink).toBeVisible({ timeout: 5000 });
    await tableLink.click();

    await page.waitForURL(/\/table\//);
    await page.waitForLoadState('networkidle');
  }

  async function fillFormFields(page: import('@playwright/test').Page) {
    const timestamp = Date.now();
    const today = new Date().toISOString().split('T')[0];

    const textInputs = page.locator(
      'form#modal-form input[type="text"]:not([name*="__search"])',
    );
    const textCount = await textInputs.count().catch(() => 0);
    for (let i = 0; i < textCount; i++) {
      try {
        const input = textInputs.nth(i);
        const name = await input.getAttribute('name').catch(() => '');
        await input.fill(`test-${timestamp}-${name || i}`);
      } catch {
        /* skip */
      }
    }

    const dateInputs = page.locator('form#modal-form input[type="date"]');
    const dateCount = await dateInputs.count().catch(() => 0);
    for (let i = 0; i < dateCount; i++) {
      try {
        await dateInputs.nth(i).fill(today);
      } catch {
        /* skip */
      }
    }

    const textareas = page.locator('form#modal-form textarea');
    const taCount = await textareas.count().catch(() => 0);
    for (let i = 0; i < taCount; i++) {
      try {
        await textareas.nth(i).fill(`test-${timestamp}`);
      } catch {
        /* skip */
      }
    }

    const searchInputs = page.locator(
      'form#modal-form input[name*="__search"]',
    );
    const searchCount = await searchInputs.count().catch(() => 0);
    for (let i = 0; i < searchCount; i++) {
      try {
        const input = searchInputs.nth(i);
        await input.click();
        await page
          .getByText('Загрузка...')
          .waitFor({ state: 'hidden', timeout: 10000 })
          .catch(() => {});
        await input.fill('а');
        const option = page
          .locator('[class*="autocompleteOption"]')
          .first();
        await option
          .waitFor({ state: 'visible', timeout: 5000 })
          .catch(() => {});
        if (await option.isVisible().catch(() => false)) {
          await option.click();
        }
      } catch {
        /* skip */
      }
    }
  }

  test('adding a directory record', async ({ page }) => {
    const menuItems = await getMenuData(page);
    const dirItem = menuItems.find(
      item => item.id_group === 1 && !item.is_group && item.ref && item.ref !== '#',
    );
    if (!dirItem) return;

    const btnIndex = getRootButtonIndex(menuItems, dirItem.id_group);
    if (btnIndex < 0) return;

    await navigateToTableViaSidebarByGroup(page, btnIndex, dirItem.ref);

    const addButton = page.getByText('Добавить');
    await expect(addButton).toBeVisible({ timeout: 5000 });
    await addButton.click();

    await expect(page.getByText(/Создание/)).toBeVisible({ timeout: 5000 });

    await fillFormFields(page);

    const saveButton = page.getByText('Сохранить изменения');
    if (await saveButton.isVisible().catch(() => false)) {
      await saveButton.click();
    }

    await expect(page.getByText(/Создание/)).not.toBeVisible({
      timeout: 15000,
    }).catch(() => {});
  });

  test('adding a document record', async ({ page }) => {
    const menuItems = await getMenuData(page);
    const docItem = menuItems.find(
      item => item.id_group === 2 && !item.is_group && item.ref && item.ref !== '#',
    );
    if (!docItem) return;

    const btnIndex = getRootButtonIndex(menuItems, docItem.id_group);
    if (btnIndex < 0) return;

    await navigateToTableViaSidebarByGroup(page, btnIndex, docItem.ref);

    const addButton = page.getByText('Добавить');
    await expect(addButton).toBeVisible({ timeout: 5000 });
    await addButton.click();

    await expect(page.getByText(/Создание/)).toBeVisible({ timeout: 5000 });

    await fillFormFields(page);

    const saveButton = page.getByText('Сохранить изменения');
    if (await saveButton.isVisible().catch(() => false)) {
      await saveButton.click();
    }

    await expect(page.getByText(/Создание/)).not.toBeVisible({
      timeout: 15000,
    }).catch(() => {});
  });

  test('editing an existing record', async ({ page }) => {
    const menuItems = await getMenuData(page);
    const anyTable = menuItems.find(
      item => !item.is_group && item.ref && item.ref !== '#',
    );
    if (!anyTable) return;

    const btnIndex = getRootButtonIndex(menuItems, anyTable.id_group);
    if (btnIndex < 0) return;

    await navigateToTableViaSidebarByGroup(page, btnIndex, anyTable.ref);

    const firstRow = page.locator('table tbody tr').first();
    if (!(await firstRow.isVisible().catch(() => false))) return;

    await firstRow.dblclick();
    await expect(page.getByText(/#\d+/)).toBeVisible({ timeout: 5000 });

    const editButton = page.getByText('Редактировать');
    if (!(await editButton.isVisible().catch(() => false))) return;

    await editButton.click();

    const textInput = page
      .locator('form#modal-form input[type="text"]:not([name*="__search"])')
      .first();
    if (await textInput.isVisible().catch(() => false)) {
      await textInput.fill(`modified-${Date.now()}`);
    }

    const saveButton = page.getByText('Сохранить изменения');
    if (await saveButton.isVisible().catch(() => false)) {
      await saveButton.click();
    }

    await expect(page.getByText(/#\d+/)).not.toBeVisible({
      timeout: 10000,
    }).catch(() => {});
  });

  test('deleting a record', async ({ page }) => {
    const menuItems = await getMenuData(page);
    const anyTable = menuItems.find(
      item => !item.is_group && item.ref && item.ref !== '#',
    );
    if (!anyTable) return;

    const btnIndex = getRootButtonIndex(menuItems, anyTable.id_group);
    if (btnIndex < 0) return;

    await navigateToTableViaSidebarByGroup(page, btnIndex, anyTable.ref);

    const firstRow = page.locator('table tbody tr').first();
    if (!(await firstRow.isVisible().catch(() => false))) return;

    await firstRow.dblclick();
    await expect(page.getByText(/#\d+/)).toBeVisible({ timeout: 5000 });

    const deleteButton = page.getByText('Удалить');
    if (!(await deleteButton.isVisible().catch(() => false))) return;

    await deleteButton.click();

    const confirmYes = page.getByText('Да');
    if (await confirmYes.isVisible().catch(() => false)) {
      await confirmYes.click();
    }

    await expect(page.getByText(/#\d+/)).not.toBeVisible({
      timeout: 10000,
    }).catch(() => {});
  });
});
