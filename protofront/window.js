// DesktopWindow: composer (single responsibility: orchestrate frame +
// table + item form + toolbar actions). Rendering lives in TableView,
// the modal in ItemForm, chrome in WindowFrame.
class DesktopWindow {
  constructor(options = {}) {
    this.title = options.title || "Window";
    this.data = options.data || [];
    this.columns = options.columns || [];
    this.onSave = options.onSave || null;
    this.onDelete = options.onDelete || null;
    this.onEdit = options.onEdit || null;
    this.onAccept = options.onAccept || null;
    this.onClose = options.onClose || null;
    this.parentContainer =
      options.container || document.getElementById("desktop");
    this.element = null;
    this.currentFormData = {};
    this.itemTitle = options.itemTitle || "Элемент справочника";
    this.itemFields = options.itemFields || options.columns || [];
    this.itemBlocks = options.itemBlocks || [];
    this.systemObjectId = options.systemObjectId || null;
    this.systemObjectGroupId = options.systemObjectGroupId || null;
    this.frame = new WindowFrame({ title: this.title });
    this.tableView = null;
    this.itemWindow = null;
    this.bounds = {
      width: options.width || 600,
      height: options.height || 400,
      left: options.left || 100,
      top: options.top || 100,
    };
  }

  render() {
    const content = this.frame.mount(this.parentContainer, this.bounds);
    this.taskbarId =
      `task-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
    const taskbar = TaskbarManager.global();
    if (taskbar) {
      taskbar.track({
        id: this.taskbarId,
        title: this.title,
        icon: "📄",
        isMinimized: () => this.frame.isMinimized(),
        minimize: () => this.frame.minimize(),
        restore: () => this.frame.restore(),
      });
      this.frame.onMinimize = () => taskbar.refreshById(this.taskbarId);
      this.frame.onRestore = () => taskbar.refreshById(this.taskbarId);
    }
    this.frame.onClose = () => {
      if (taskbar) taskbar.untrack(this.taskbarId);
      if (this.itemWindow) this.itemWindow.destroy();
      if (this.onClose) this.onClose();
    };
    this.element = this.frame.element;

    const toolbar = document.createElement("div");
    toolbar.className = "window-toolbar";
    toolbar.innerHTML = `
      <button class="btn-add btn-window">➕ Добавить</button>
      <button class="btn-edit btn-window" disabled>✏️ Редактировать</button>
      <button class="btn-delete btn-window" disabled>🗑️ Удалить</button>
      <button class="btn-approve btn-window" disabled>✅ Провести</button>
      <button class="btn-related btn-window" disabled>🔗 Ввести на основании</button>
      <button class="btn-refresh btn-window">🔄 Обновить</button>
    `;
    content.appendChild(toolbar);

    const tableContainer = document.createElement("div");
    tableContainer.className = "table-container";
    this.tableView = new TableView(this.columns, {
      onEditItem: (item) => this.editItem(item),
    });
    const table = this.tableView.createTable();
    tableContainer.appendChild(table);
    content.appendChild(tableContainer);
    this.renderTableRows();

    this.initItemWindow();
    this.attachEvents();
    this.frame.bringToFront();
    return this.element;
  }

  initItemWindow() {
    const self = this;
    this.itemWindow = new ItemForm({
      fields: this.itemFields,
      blocks: this.itemBlocks,
      itemTitle: this.itemTitle,
      onSave: async (newItem) => {
        this.data.push(newItem);
        if (this.onSave) {
          await this.onSave(newItem);
        }
        this.refreshData();
      },
      onEdit: async (editedItem) => {
        const index = this.data.findIndex(
          (item) => (item.id || item.id) === editedItem.id,
        );
        if (index !== -1) {
          this.data[index] = { ...this.data[index], ...editedItem };
          if (this.onEdit) {
            await this.onEdit(this.data[index]);
          }
        }
        this.refreshData();
      },
      onAccept: async (acceptItem) => {
        const index = this.data.findIndex(
          (item) => (item.id || item.id) === acceptItem.id,
        );
        if (index !== -1) {
          this.data[index] = { ...this.data[index], ...acceptItem };
          if (this.onAccept) {
            await this.onAccept(this.data[index]);
          }
        }
        this.refreshData();
      },
      parentWindow: {
        // Adapter: ItemForm needs a mount point + z-order bump.
        // `element` is live-read so the modal follows re-renders.
        get element() {
          return self.element;
        },
        bringToFront: () => self.frame.bringToFront(),
      },
      systemObjectGroupId: this.systemObjectGroupId,
    });
  }

  renderTableRows() {
    const tbody = this.element.querySelector(".data-table tbody");
    return this.tableView.renderRows(tbody, this.data);
  }

  attachEvents() {
    const toolbar = this.element.querySelector(".window-toolbar");
    toolbar.querySelector(".btn-add").addEventListener("click", () => this.showForm());
    toolbar.querySelector(".btn-edit").addEventListener("click", () => this.editSelected());
    toolbar.querySelector(".btn-delete").addEventListener("click", () => this.deleteSelected());
    toolbar.querySelector(".btn-approve").addEventListener("click", () => this.approveSelected());
    toolbar.querySelector(".btn-related").addEventListener("click", () => this.relatedSelected());
    toolbar.querySelector(".btn-refresh").addEventListener("click", () => this.refreshData());

    const selectAll = this.element.querySelector(".select-all");
    if (selectAll) {
      selectAll.addEventListener("change", (e) =>
        this.selectAllRows(e.target.checked),
      );
    }
  }

  showForm() {
    if (this.itemWindow) {
      this.itemWindow.showForm(null, this.element);
    }
  }

  editSelected() {
    const selected = this.getSelectedRows();
    if (selected.length !== 1) {
      alert("Пожалуйста, выберите одну запись для редактирования");
      return;
    }
    const item = this.data.find((d) => d.id === selected[0]);
    if (item && this.itemWindow) {
      this.itemWindow.editItem(item, this.element);
    }
  }

  editItem(item) {
    if (item && this.itemWindow) {
      this.itemWindow.editItem(item, this.element);
    }
  }

  async deleteSelected() {
    const selected = this.getSelectedRows();
    if (selected.length === 0) {
      alert("Пожалуйста, выберите записи для удаления");
      return;
    }

    if (confirm(`Удалить ${selected.length} запись(и)?`)) {
      const deleteItems = this.data.filter((item) =>
        selected.includes(item.id),
      );
      this.data = this.data.filter((item) => !selected.includes(item.id));
      if (this.onDelete) {
        for (const item of deleteItems) {
          await this.onDelete(item.id);
        }
      }
      this.refreshData();
    }
  }

  async approveSelected() {
    const selected = this.getSelectedRows();
    if (selected.length !== 1) {
      alert("Пожалуйста, выберите одну запись для проведения");
      return;
    }
    const item = this.data.find((d) => d.id === selected[0]);
    if (item && this.onAccept) {
      await this.onAccept(item);
      this.refreshData();
    }
  }

  relatedSelected() {
    console.warn("DesktopWindow: ввод на основании не реализован");
  }

  getSelectedRows() {
    return this.tableView.getSelectedIds(this.element);
  }

  selectAllRows(checked) {
    this.tableView.setAllSelected(this.element, checked);
    this.updateToolbarButtons();
  }

  updateToolbarButtons() {
    const selectedCount = this.getSelectedRows().length;
    const editBtn = this.element.querySelector(".btn-edit");
    const deleteBtn = this.element.querySelector(".btn-delete");

    if (editBtn) editBtn.disabled = selectedCount !== 1;
    if (deleteBtn) deleteBtn.disabled = selectedCount === 0;
  }

  async refreshData() {
    await this.renderTableRows();

    const checkboxes = this.element.querySelectorAll(".row-select");
    checkboxes.forEach((cb) => {
      cb.addEventListener("change", () => this.updateToolbarButtons());
    });

    const addBtn = this.element.querySelector(".btn-add");
    const editBtn = this.element.querySelector(".btn-edit");
    const deleteBtn = this.element.querySelector(".btn-delete");
    if (this.systemObjectId > 5 && this.systemObjectId < 12) {
      addBtn.disabled = true;
      editBtn.disabled = true;
      deleteBtn.disabled = true;
    }
    this.updateToolbarButtons();
  }

  maximize() {
    this.frame.maximize();
  }

  close() {
    if (this.itemWindow) {
      this.itemWindow.destroy();
    }
    this.frame.close();
  }
}

// Export for use
if (typeof module !== "undefined" && module.exports) {
  module.exports = DesktopWindow;
}
