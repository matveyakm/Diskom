class EnhancedSelect {
  constructor(container, options = {}) {
    if (!container) {
      throw new Error("Container element is required");
    }

    this.container = container;
    this.options = {
      items: [],
      multiple: false,
      searchable: true,
      placeholder: "-- Выберите --",
      emptyMessage: "Нет элементов",
      onChange: () => {},
      ...options,
    };

    this.items = [...this.options.items];
    this.filteredItems = [...this.items];
    this.selectedItems = [];

    this.init();
  }

  init() {
    this.render();
    this.attachEvents();
  }

  render() {
    this.container.innerHTML = "";

    const wrapper = document.createElement("div");
    wrapper.className = "enhanced-select";

    if (this.options.searchable) {
      const searchInput = document.createElement("input");
      searchInput.type = "text";
      searchInput.placeholder = "🔍 Поиск...";
      searchInput.className = "select-search";
      wrapper.appendChild(searchInput);
      this.searchInput = searchInput;
    }

    const selectContainer = document.createElement("div");
    selectContainer.className = "select-container";

    const select = document.createElement("select");
    select.className = "custom-select";

    if (this.options.multiple) {
      select.multiple = true;
    }

    wrapper.appendChild(selectContainer);
    selectContainer.appendChild(select);
    this.container.appendChild(wrapper);

    this.select = select;
    this.updateSelectOptions();
  }

  attachEvents() {
    if (this.searchInput) {
      this.searchInput.addEventListener("input", (e) => {
        this.filterItems(e.target.value);
      });
    }

    if (this.select) {
      this.select.addEventListener("change", () => {
        this.handleSelectionChange();
      });
    }
  }

  updateSelectOptions() {
    if (!this.select) return;

    const currentValues = Array.from(this.select.selectedOptions).map(
      (opt) => opt.value,
    );
    this.select.innerHTML = "";

    if (!this.options.multiple) {
      const defaultOption = document.createElement("option");
      defaultOption.value = "";
      defaultOption.textContent = this.options.placeholder;
      this.select.appendChild(defaultOption);
    }

    if (this.filteredItems.length === 0) {
      const emptyOption = document.createElement("option");
      emptyOption.value = "";
      emptyOption.textContent = this.options.emptyMessage;
      emptyOption.disabled = true;
      this.select.appendChild(emptyOption);
    } else {
      this.filteredItems.forEach((item) => {
        const option = document.createElement("option");
        option.value = item.id;
        option.textContent =
          item.name || item.label || item.title || `Элемент ${item.id}`;

        if (currentValues.includes(String(item.id))) {
          option.selected = true;
        }

        this.select.appendChild(option);
      });
    }

    // Обновляем размер для множественного выбора
    if (this.options.multiple) {
      const size = Math.min(this.filteredItems.length || 1, 10);
      this.select.size = size;
    }
  }

  filterItems(searchTerm) {
    if (!searchTerm || searchTerm.trim() === "") {
      this.filteredItems = [...this.items];
    } else {
      const term = searchTerm.toLowerCase();
      this.filteredItems = this.items.filter((item) => {
        const itemName = (
          item.name ||
          item.label ||
          item.title ||
          ""
        ).toLowerCase();
        return itemName.includes(term);
      });
    }

    this.updateSelectOptions();
  }

  handleSelectionChange() {
    if (!this.select) return;

    const selectedOptions = Array.from(this.select.selectedOptions);
    const selectedIds = selectedOptions
      .map((opt) => opt.value)
      .filter((v) => v);

    if (this.options.multiple) {
      this.selectedItems = this.items.filter((item) =>
        selectedIds.includes(String(item.id)),
      );
    } else {
      const selectedId = selectedIds[0];
      this.selectedItems = selectedId
        ? [this.items.find((item) => String(item.id) === selectedId)]
        : [];
    }

    const result = this.options.multiple
      ? this.selectedItems
      : this.selectedItems[0] || null;

    this.options.onChange(result);
  }

  getSelected() {
    return this.options.multiple
      ? this.selectedItems
      : this.selectedItems[0] || null;
  }

  setItems(items) {
    this.items = [...items];
    this.filteredItems = [...items];
    this.selectedItems = [];
    this.updateSelectOptions();
  }

  clear() {
    if (this.select) {
      this.select.selectedIndex = -1;
      if (this.select.multiple) {
        Array.from(this.select.options).forEach(
          (opt) => (opt.selected = false),
        );
      }
    }
    this.selectedItems = [];
  }
}

// ItemWindow Class for handling add/edit forms
class ItemWindow {
  constructor(options = {}) {
    this.fields = options.fields || [];
    this.onSave = options.onSave || null;
    this.onEdit = options.onEdit || null;
    this.onAccept = options.onAccept || null;
    this.parentWindow = options.parentWindow || null;
    this.element = null;
    this.editingId = null;
    this.editingItem = null;
    this.itemTitle = options.itemTitle || "Элемент справочника";
    this.selectInstances = {}; // Хранилище для select компонентов
  }

  // Create form modal
  createFormModal() {
    const modal = document.createElement("div");
    modal.className = "window-modal";
    modal.style.display = "none";

    const modalContent = document.createElement("div");
    modalContent.className = "window-modal-content";

    const modalHeader = document.createElement("div");
    modalHeader.className = "window-modal-header";
    modalHeader.innerHTML = `<h3 id="modal-title">${this.itemTitle}</h3> <button class="modal-close">&times;</button>`;

    const modalBody = document.createElement("div");
    modalBody.className = "window-modal-body";

    const form = document.createElement("form");
    form.id = "data-form";

    // Create form fields based on columns
    this.fields.forEach((col) => {
      if (col.field !== "id" && !col.hidden) {
        const formGroup = document.createElement("div");
        formGroup.className = "form-group";
        formGroup.id = `group-${col.field}`;

        const label = document.createElement("label");
        label.textContent = col.label;
        label.htmlFor = `field-${col.field}`;

        if (col.type === "textarea") {
          const input = document.createElement("textarea");
          input.rows = 3;
          input.id = `field-${col.field}`;
          input.name = col.field;
          input.className = "form-input";
          formGroup.appendChild(label);
          formGroup.appendChild(input);
        } else if (col.type === "select" && col.end_point) {
          // Создаем контейнер для select
          const selectContainer = document.createElement("div");
          selectContainer.id = `select-container-${col.field}`;
          selectContainer.className = "enhanced-select-wrapper";

          formGroup.appendChild(label);
          formGroup.appendChild(selectContainer);

          // Сохраняем информацию о select для инициализации после рендера
          this.selectInstances[col.field] = {
            container: selectContainer,
            endPoint: col.end_point,
            field: col.field,
            multiple: col.multiple || false,
          };
        } else {
          const input = document.createElement("input");
          input.type = col.type || "text";
          input.id = `field-${col.field}`;
          input.name = col.field;
          input.className = "form-input";
          input.placeholder =
            col.placeholder || `Введите ${col.label.toLowerCase()}`;
          input.required = col.required || false;
          if (col.pattern) input.pattern = col.pattern;
          if (col.field === "ddate") {
            input.type = "datetime-local";
          }

          formGroup.appendChild(label);
          formGroup.appendChild(input);
        }

        modalBody.appendChild(formGroup);
      }
    });

    const modalFooter = document.createElement("div");
    modalFooter.className = "window-modal-footer";
    modalFooter.innerHTML = `
      <button type="button" class="btn-cancel">Отмена</button>
      <button type="submit" class="btn-save">Сохранить</button>
      <button type="button" class="btn-accept">Провести</button>
    `;

    form.appendChild(modalBody);
    form.appendChild(modalFooter);
    modalContent.appendChild(modalHeader);
    modalContent.appendChild(form);
    modal.appendChild(modalContent);

    return modal;
  }

  // Initialize select components after modal is added to DOM
  async initSelectComponents() {
    for (const [field, config] of Object.entries(this.selectInstances)) {
      try {
        // Загружаем данные из API
        const response = await fetch(`/api/${config.endPoint}`);
        const data = await response.json();
        const items = data.data || data;

        // СоздаемEnhancedSelect
        new EnhancedSelect(config.container, {
          items: items,
          multiple: config.multiple,
          searchable: true,
          placeholder: "-- Выберите --",
          onChange: (selected) => {
            console.log(`Выбрано для поля ${field}:`, selected);
            // Сохраняем выбранное значение в formData
            if (this.currentFormData) {
              this.currentFormData[field] = selected;
            }
            // Также сохраняем в скрытое поле или атрибут
            if (this.element) {
              const hiddenInput = this.element.querySelector(
                `#hidden-${field}`,
              );
              if (hiddenInput) {
                hiddenInput.value = selected?.id || "";
              }
            }
          },
        });
      } catch (error) {
        console.error(`Error loading data for field ${field}:`, error);
      }
    }
  }

  // Show form for add/edit
  async showForm(itemData = null, parentElement = null) {
    if (!this.element) {
      this.element = this.createFormModal();
      if (parentElement) {
        parentElement.appendChild(this.element);
      } else if (this.parentWindow && this.parentWindow.element) {
        this.parentWindow.element.appendChild(this.element);
      } else {
        document.body.appendChild(this.element);
      }
      this.attachFormEvents();

      // Инициализируем select компоненты после добавления в DOM
      await this.initSelectComponents();
    }

    const modal = this.element;
    const title = modal.querySelector("#modal-title");
    const form = modal.querySelector("#data-form");

    this.currentFormData = {};

    if (itemData) {
      title.textContent = `Редактирование: ${this.itemTitle}`;
      this.editingId = itemData.id;
      this.editingItem = itemData;

      // Fill form with existing data
      this.fields.forEach((col) => {
        const input = form.querySelector(`#field-${col.field}`);
        if (input && itemData[col.field] !== undefined) {
          input.value = itemData[col.field];
        }
        // Для select полей
        if (col.type === "select" && this.selectInstances[col.field]) {
          this.currentFormData[col.field] = itemData[col.field];
        }
      });
    } else {
      title.textContent = `Добавление: ${this.itemTitle}`;
      this.editingId = null;
      this.editingItem = null;
      this.fields.forEach((col) => {
        if (col.field !== undefined && col.field === "ddate") {
          const input = form.querySelector(`#field-${col.field}`);
          input.value = new Date().toISOString().slice(0, 16);
        }
      });

      //      form.reset();
    }

    modal.style.display = "flex";

    // Bring to front
    if (this.parentWindow) {
      this.parentWindow.bringToFront();
    }
  }

  // Save item (add or edit)
  async saveItem(e) {
    e.preventDefault();

    const form = e.target;
    const formData = {};

    // Collect form data from regular inputs
    this.fields.forEach((col) => {
      if (col.field !== "id" && col.field !== "Id") {
        const input = form.querySelector(`#field-${col.field}`);
        if (input) {
          formData[col.field] = input.value;
        }
        // Collect from select components
        if (
          col.type === "select" &&
          this.currentFormData &&
          this.currentFormData[col.field]
        ) {
          formData[col.field] =
            this.currentFormData[col.field]?.id ||
            this.currentFormData[col.field];
        }
      }
    });

    let result = null;

    if (this.editingId !== null) {
      // Edit existing item
      if (this.onEdit) {
        result = await this.onEdit({
          id: this.editingId,
          ...formData,
        });
      }
    } else {
      // Add new item
      const newItem = {
        status: 1,
        ...formData,
      };

      if (this.onSave) {
        result = await this.onSave(newItem);
      }
    }

    this.hideForm();
    return result;
  }

  // Add this method after acceptItem():
  async acceptItem(acceptedItem) {
    if (this.onAccept) {
      acceptedItem.id = this.editingId;
      result = await this.onAccept(acceptedItem);
    }
  }

  // Edit item by ID
  editItem(item, parentElement = null) {
    if (item) {
      this.showForm(item, parentElement);
    }
  }

  // Attach form events
  attachFormEvents() {
    if (!this.element) return;

    const closeModalBtn = this.element.querySelector(".modal-close");
    const cancelBtn = this.element.querySelector(".btn-cancel");
    const acceptBtn = this.element.querySelector(".btn-accept");
    const form = this.element.querySelector("#data-form");

    if (closeModalBtn) {
      closeModalBtn.addEventListener("click", () => this.hideForm());
    }

    if (cancelBtn) {
      cancelBtn.addEventListener("click", () => this.hideForm());
    }

    if (form) {
      form.addEventListener("submit", (e) => this.saveItem(e));
    }

    if (acceptBtn) {
      acceptBtn.addEventListener("click", (e) => this.acceptItem(e));
    }
  }

  // Hide form
  hideForm() {
    if (this.element) {
      this.element.style.display = "none";
      this.editingId = null;
      this.editingItem = null;
      this.currentFormData = null;
    }
  }

  // Close and remove form
  destroy() {
    if (this.element) {
      this.element.remove();
      this.element = null;
    }
  }
}

// Window Manager Class
class DesktopWindow {
  constructor(options = {}) {
    this.id =
      options.id ||
      `window-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
    this.title = options.title || "Window";
    this.width = options.width || 600;
    this.height = options.height || 400;
    this.left = options.left || 100;
    this.top = options.top || 100;
    this.data = options.data || [];
    this.columns = options.columns || [];
    this.onSave = options.onSave || null;
    this.onDelete = options.onDelete || null;
    this.onEdit = options.onEdit || null;
    this.onAccept = options.onAccept || null;
    this.parentContainer =
      options.container || document.getElementById("desktop");
    this.element = null;
    this.isDragging = false;
    this.dragStart = { x: 0, y: 0 };
    this.currentFormData = {};
    this.editingId = null;
    this.itemWindow = null; // Reference to ItemWindow instance
    this.itemTitle = options.itemTitle || "Элемент справочника";
    this.itemFields = options.itemFields || options.columns || [];
    this.systemObjectId = options.systemObjectId || null;
    this.systemObjectGroupId = options.systemObjectGroupId || null;
  }

  // Create window HTML structure
  render() {
    // Create window container
    const windowDiv = document.createElement("div");
    windowDiv.className = "desktop-window";
    windowDiv.id = this.id;
    windowDiv.style.width = `${this.width}px`;
    windowDiv.style.height = `${this.height}px`;
    windowDiv.style.left = `${this.left}px`;
    windowDiv.style.top = `${this.top}px`;
    windowDiv.style.zIndex = this.getHighestZIndex() + 1;

    // Window header
    const header = document.createElement("div");
    header.className = "window-header";
    header.innerHTML = `
      <div class="window-title">
        <span class="window-title-icon">📄</span>
        <span>${this.escapeHtml(this.title)}</span>
      </div>
      <div class="window-controls">
        <button class="window-btn window-minimize" title="Свернуть">─</button>
        <button class="window-btn window-maximize" title="Развернуть">□</button>
        <button class="window-btn window-close" title="Закрыть">✕</button>
      </div>
    `;

    // Window content
    const content = document.createElement("div");
    content.className = "window-content";

    // Toolbar
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

    // Table container
    const tableContainer = document.createElement("div");
    tableContainer.className = "table-container";
    const table = this.createTable();
    tableContainer.appendChild(table);
    content.appendChild(tableContainer);

    windowDiv.appendChild(header);
    windowDiv.appendChild(content);
    this.parentContainer.appendChild(windowDiv);
    this.element = windowDiv;

    // Initialize ItemWindow
    this.initItemWindow();

    this.attachEvents();
    this.makeDraggable();
    this.makeResizable();
    this.bringToFront();

    return windowDiv;
  }

  // Initialize ItemWindow instance
  initItemWindow() {
    this.itemWindow = new ItemWindow({
      fields: this.itemFields,
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
      parentWindow: this,
      systemObjectGroupId: this.systemObjectGroupId,
    });
  }

  // Create data table
  createTable() {
    const table = document.createElement("table");
    table.className = "data-table";

    // Create header
    const thead = document.createElement("thead");
    const headerRow = document.createElement("tr");

    // Add checkbox column for selection
    const selectTh = document.createElement("th");
    selectTh.innerHTML = '<input type="checkbox" class="select-all">';
    selectTh.style.width = "40px";
    headerRow.appendChild(selectTh);

    // Add data columns
    this.columns.forEach((col) => {
      const th = document.createElement("th");
      th.textContent = col.label;
      if (col.width) th.style.width = col.width;
      headerRow.appendChild(th);
    });

    thead.appendChild(headerRow);
    table.appendChild(thead);

    // Create body
    const tbody = document.createElement("tbody");
    this.renderTableRows(tbody);
    table.appendChild(tbody);

    return table;
  }

  // Render table rows
  async renderTableRows(tbody) {
    tbody.innerHTML = "";

    if (this.data.length === 0) {
      const emptyRow = document.createElement("tr");
      emptyRow.className = "empty-row";
      const emptyCell = document.createElement("td");
      emptyCell.colSpan = this.columns.length + 1;
      emptyCell.textContent = "Нет данных";
      emptyCell.style.textAlign = "center";
      emptyCell.style.padding = "40px";
      emptyRow.appendChild(emptyCell);
      tbody.appendChild(emptyRow);
      return;
    }

    for (const item of this.data) {
      const row = document.createElement("tr");
      row.dataset.id = item.id;

      // Checkbox cell
      const selectCell = document.createElement("td");
      const checkbox = document.createElement("input");
      checkbox.type = "checkbox";
      checkbox.className = "row-select";
      checkbox.dataset.id = item.id;
      selectCell.appendChild(checkbox);
      row.appendChild(selectCell);

      // Data cells
      for (const col of this.columns) {
        const cell = document.createElement("td");

        if (col.field === "status") {
          cell.classList.add("status-cell");
          cell.setAttribute("data-value", item[col.field]);
          cell.textContent = ""; // Clear text content
        } else if (col.end_point) {
          // Асинхронная загрузка связанных данных
          try {
            const response = await fetch(
              `/api/${col.end_point}/${item[col.field]}`,
            );
            const data = await response.json();
            cell.textContent = data.name || data.label || "-";
          } catch (error) {
            cell.textContent = "-";
          }
        } else {
          cell.textContent = item[col.field] || "-";
        }

        row.appendChild(cell);
      }

      // Double-click to edit
      row.addEventListener("dblclick", () => this.editItem(item));

      tbody.appendChild(row);
    }
  }

  // Attach event listeners
  attachEvents() {
    const toolbar = this.element.querySelector(".window-toolbar");
    const addBtn = toolbar.querySelector(".btn-add");
    const editBtn = toolbar.querySelector(".btn-edit");
    const deleteBtn = toolbar.querySelector(".btn-delete");
    const approveBtn = toolbar.querySelector(".btn-approve");
    const relatedBtn = toolbar.querySelector(".btn-related");
    const refreshBtn = toolbar.querySelector(".btn-refresh");

    addBtn.addEventListener("click", () => this.showForm());
    editBtn.addEventListener("click", () => this.editSelected());
    deleteBtn.addEventListener("click", () => this.deleteSelected());
    approveBtn.addEventListener("click", () => this.approveSelected());
    relatedBtn.addEventListener("click", () => this.relatedSelected());
    refreshBtn.addEventListener("click", () => this.refreshData());

    // Window controls
    const controls = this.element.querySelector(".window-controls");
    const minimizeBtn = controls.querySelector(".window-minimize");
    const maximizeBtn = controls.querySelector(".window-maximize");
    const closeBtn = controls.querySelector(".window-close");

    minimizeBtn.addEventListener("click", () => this.minimize());
    maximizeBtn.addEventListener("click", () => this.maximize());
    closeBtn.addEventListener("click", () => this.close());

    // Select all checkbox
    const selectAll = this.element.querySelector(".select-all");
    if (selectAll) {
      selectAll.addEventListener("change", (e) =>
        this.selectAllRows(e.target.checked),
      );
    }
  }

  // Show form for add
  showForm() {
    if (this.itemWindow) {
      this.itemWindow.showForm(null, this.element);
    }
  }

  // Edit selected item
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

  // Edit item by item object
  editItem(item) {
    if (item && this.itemWindow) {
      this.itemWindow.editItem(item, this.element);
    }
  }

  // Delete selected items
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
        deleteItems.forEach(async (item) => {
          await this.onDelete(item.id);
        });
      }
      this.refreshData();
    }
  }

  // Get selected row IDs
  getSelectedRows() {
    const checkboxes = this.element.querySelectorAll(".row-select:checked");
    return Array.from(checkboxes).map((cb) => {
      const id = cb.dataset.id;
      return isNaN(id) ? id : parseInt(id);
    });
  }

  // Select all rows
  selectAllRows(checked) {
    const checkboxes = this.element.querySelectorAll(".row-select");
    checkboxes.forEach((cb) => (cb.checked = checked));
    this.updateToolbarButtons();
  }

  // Update toolbar buttons state
  updateToolbarButtons() {
    const selectedCount = this.getSelectedRows().length;
    const editBtn = this.element.querySelector(".btn-edit");
    const deleteBtn = this.element.querySelector(".btn-delete");

    if (editBtn) editBtn.disabled = selectedCount !== 1;
    if (deleteBtn) deleteBtn.disabled = selectedCount === 0;
  }

  // Refresh table data
  refreshData() {
    const tbody = this.element.querySelector(".data-table tbody");
    this.renderTableRows(tbody);

    // Reattach row selection events
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

  // Make window draggable
  makeDraggable() {
    const header = this.element.querySelector(".window-header");

    header.addEventListener("mousedown", (e) => {
      if (e.target.closest(".window-controls")) return;

      this.isDragging = true;
      this.dragStart.x = e.clientX - this.element.offsetLeft;
      this.dragStart.y = e.clientY - this.element.offsetTop;
      this.element.style.cursor = "grabbing";
      this.bringToFront();

      document.addEventListener("mousemove", this.onDrag.bind(this));
      document.addEventListener("mouseup", this.stopDrag.bind(this));
    });
  }

  onDrag(e) {
    if (!this.isDragging) return;

    let newLeft = e.clientX - this.dragStart.x;
    let newTop = e.clientY - this.dragStart.y;

    newLeft = Math.max(
      0,
      Math.min(newLeft, window.innerWidth - this.element.offsetWidth),
    );
    newTop = Math.max(
      0,
      Math.min(newTop, window.innerHeight - this.element.offsetHeight),
    );

    this.element.style.left = `${newLeft}px`;
    this.element.style.top = `${newTop}px`;
  }

  stopDrag() {
    this.isDragging = false;
    this.element.style.cursor = "";
    document.removeEventListener("mousemove", this.onDrag);
    document.removeEventListener("mouseup", this.stopDrag);
  }

  // Make window resizable
  makeResizable() {
    const resizeHandle = document.createElement("div");
    resizeHandle.className = "window-resize-handle";
    this.element.appendChild(resizeHandle);

    let isResizing = false;
    let startX, startY, startWidth, startHeight;

    resizeHandle.addEventListener("mousedown", (e) => {
      isResizing = true;
      startX = e.clientX;
      startY = e.clientY;
      startWidth = this.element.offsetWidth;
      startHeight = this.element.offsetHeight;

      document.addEventListener("mousemove", onMouseMove);
      document.addEventListener("mouseup", onMouseUp);
    });

    const onMouseMove = (e) => {
      if (!isResizing) return;

      const newWidth = Math.max(300, startWidth + (e.clientX - startX));
      const newHeight = Math.max(200, startHeight + (e.clientY - startY));

      this.element.style.width = `${newWidth}px`;
      this.element.style.height = `${newHeight}px`;
    };

    const onMouseUp = () => {
      isResizing = false;
      document.removeEventListener("mousemove", onMouseMove);
      document.removeEventListener("mouseup", onMouseUp);
    };
  }

  // Minimize window
  minimize() {
    this.element.classList.add("minimized");
    setTimeout(() => {
      this.element.style.display = "none";
    }, 300);
  }

  // Maximize window
  maximize() {
    if (this.element.classList.contains("maximized")) {
      this.element.classList.remove("maximized");
      this.element.style.width = `${this.width}px`;
      this.element.style.height = `${this.height}px`;
      this.element.style.left = `${this.left}px`;
      this.element.style.top = `${this.top}px`;
    } else {
      this.element.classList.add("maximized");
      this.element.style.width = "100%";
      this.element.style.height = `calc(100vh - 48px)`;
      this.element.style.left = "0";
      this.element.style.top = "0";
    }
  }

  // Close window
  close() {
    if (this.itemWindow) {
      this.itemWindow.destroy();
    }
    this.element.classList.add("closing");
    setTimeout(() => {
      this.element.remove();
      if (this.onClose) this.onClose();
    }, 300);
  }

  // Bring window to front
  bringToFront() {
    const windows = document.querySelectorAll(".desktop-window");
    let maxZIndex = 0;
    windows.forEach((win) => {
      const zIndex = parseInt(win.style.zIndex);
      if (zIndex > maxZIndex) maxZIndex = zIndex;
    });
    this.element.style.zIndex = maxZIndex + 1;
  }

  getHighestZIndex() {
    const windows = document.querySelectorAll(".desktop-window");
    let maxZIndex = 1000;
    windows.forEach((win) => {
      const zIndex = parseInt(win.style.zIndex);
      if (zIndex > maxZIndex) maxZIndex = zIndex;
    });
    return maxZIndex;
  }

  escapeHtml(text) {
    const div = document.createElement("div");
    div.textContent = text;
    return div.innerHTML;
  }
}

const windowStyles = `
  .desktop-window {
    position: absolute;
    background: rgba(32, 32, 32, 0.95);
    backdrop-filter: blur(20px);
    border: 1px solid rgba(255, 255, 255, 0.2);
    border-radius: 12px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
    display: flex;
    flex-direction: column;
    overflow: hidden;
    transition: all 0.2s ease;
    min-width: 300px;
    min-height: 200px;
  }

  .desktop-window.maximized {
    border-radius: 0;
  }

  .desktop-window.minimized {
    transform: scale(0.9);
    opacity: 0;
  }

  .desktop-window.closing {
    transform: scale(0.8);
    opacity: 0;
  }

  .window-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 16px;
    background: rgba(255, 255, 255, 0.1);
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
    cursor: move;
    user-select: none;
  }

  .window-title {
    display: flex;
    align-items: center;
    gap: 8px;
    color: white;
    font-size: 14px;
    font-weight: 500;
  }

  .window-controls {
    display: flex;
    gap: 8px;
  }

  .window-btn {
    background: none;
    border: none;
    color: white;
    cursor: pointer;
    width: 28px;
    height: 28px;
    border-radius: 6px;
    font-size: 14px;
    transition: all 0.2s;
  }

  .window-btn:hover {
    background: rgba(255, 255, 255, 0.2);
  }

  .window-close:hover {
    background: #e81123;
  }

  .window-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    padding: 16px;
  }

  .window-toolbar {
    display: flex;
    gap: 8px;
    margin-bottom: 16px;
    padding-bottom: 12px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  }

  .btn-window {
    background: rgba(255, 255, 255, 0.1);
    border: 1px solid rgba(255, 255, 255, 0.2);
    color: white;
    padding: 6px 12px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 13px;
    transition: all 0.2s;
  }

  .btn-window:hover:not(:disabled) {
    background: rgba(255, 255, 255, 0.2);
    transform: translateY(-1px);
  }

  .btn-window:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .table-container {
    flex: 1;
    overflow: auto;
    background: rgba(0, 0, 0, 0.3);
    border-radius: 8px;
  }

  .data-table {
    width: 100%;
    border-collapse: collapse;
    color: white;
  }

  .data-table thead {
    background: rgba(50, 50, 50, 0.9);
    position: sticky;
    top: 0;
  }

  .data-table th,
  .data-table td {
    padding: 10px 12px;
    text-align: left;
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  }

  .data-table tbody tr:hover {
    background: rgba(255, 255, 255, 0.05);
    cursor: pointer;
  }

  .data-table tbody tr.selected {
    background: rgba(255, 255, 255, 0.1);
  }

  .window-modal {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 10000;
  }

  .window-modal-content {
    background: rgba(32, 32, 32, 0.95);
    backdrop-filter: blur(20px);
    border-radius: 12px;
    min-width: 400px;
    max-width: 500px;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  }

  .window-modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  }

  .window-modal-header h3 {
    margin: 0;
    color: white;
    font-size: 16px;
  }

  .modal-close {
    background: none;
    border: none;
    color: white;
    font-size: 24px;
    cursor: pointer;
    padding: 0;
    width: 30px;
    height: 30px;
    border-radius: 6px;
  }

  .modal-close:hover {
    background: rgba(255, 255, 255, 0.2);
  }

  .window-modal-body {
    padding: 20px;
  }

  .form-group {
    margin-bottom: 16px;
  }

  .form-group label {
    display: block;
    color: white;
    margin-bottom: 6px;
    font-size: 13px;
    font-weight: 500;
  }

  .form-group input,
  .form-group select,
  .form-group textarea {
    width: 100%;
    padding: 8px 12px;
    background: rgba(255, 255, 255, 0.1);
    border: 1px solid rgba(255, 255, 255, 0.2);
    border-radius: 6px;
    color: white;
    font-size: 14px;
    box-sizing: border-box;
  }

  .form-group input:focus,
  .form-group select:focus,
  .form-group textarea:focus {
    outline: none;
    border-color: #0078d4;
  }

  .window-modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    padding: 16px 20px;
    border-top: 1px solid rgba(255, 255, 255, 0.1);
  }

  .window-modal-footer button {
    padding: 8px 16px;
    border-radius: 6px;
    border: none;
    cursor: pointer;
    font-size: 13px;
    transition: all 0.2s;
  }

  .btn-cancel {
    background: rgba(255, 255, 255, 0.1);
    color: white;
  }

  .btn-cancel:hover {
    background: rgba(255, 255, 255, 0.2);
  }

  .btn-save {
    background: #0078d4;
    color: white;
  }

  .btn-save:hover {
    background: #005a9e;
  }

  .window-resize-handle {
    position: absolute;
    bottom: 0;
    right: 0;
    width: 15px;
    height: 15px;
    cursor: se-resize;
    background: linear-gradient(135deg, transparent 50%, rgba(255, 255, 255, 0.3) 50%);
    border-bottom-right-radius: 12px;
  }

  .status-cell {
    width: 8px;
    height: 8px;
    color: transparent; /* Текст станет невидимым */
    overflow: hidden;    /* На всякий случай */
    user-select: none;   /* Чтобы нельзя было случайно выделить цифру */
    filter: invert(1)
  }

  /* Если значение 1 — иконка листика (используем Unicode) */
  .status-cell[data-value="1"] {
    background-image: url('png/file.png');
    background-repeat: no-repeat;
    background-position: center;
    background-size: contain;
  }
  /* Если значение 0 — иконка с крестиком */
  .status-cell[data-value="0"] {
    background-image: url('png/delete-document.png');
    background-repeat: no-repeat;
    background-position: center;
    background-size: contain;  }

    .status-cell[data-value="2"] {
    background-image: url('png/assept-document.png');
    background-repeat: no-repeat;
    background-position: center;
    background-size: contain;  }

    /* Современные стили для select */
    .custom-select {
        width: 100%;
        padding: 10px;
        border: 2px solid #e0e0e0;
        border-radius: 8px;
        font-size: 16px;
        background-color: white;
        cursor: pointer;
        transition: all 0.3s ease;
    }

    .custom-select:hover {
        border-color: #007bff;
    }

    .custom-select:focus {
        outline: none;
        border-color: #007bff;
        box-shadow: 0 0 0 3px rgba(0,123,255,0.1);
    }

    .select-search {
        width: 100%;
        padding: 8px;
        margin-bottom: 10px;
        border: 1px solid #ddd;
        border-radius: 4px;
        font-size: 14px;
    }

    .enhanced-select {
        margin: 20px 0;
    }

    /* Для множественного выбора */
    .custom-select[multiple] {
        min-height: 150px;
    }

    .custom-select option {
        padding: 8px;
        margin: 2px 0;
    }

    .custom-select option:checked {
        background: #007bff linear-gradient(0deg, #007bff 0%, #007bff 100%);
        color: white;
    }
    .form-input {
      width: 100%;
      padding: 8px 12px;
      background: rgba(255, 255, 255, 0.1);
      border: 1px solid rgba(255, 255, 255, 0.2);
      border-radius: 6px;
      color: white;
      font-size: 14px;
      box-sizing: border-box;
    }

    .form-input:focus {
      outline: none;
      border-color: #0078d4;
    }

    .enhanced-select-wrapper {
      margin-top: 5px;
    }
`;

// Add styles to document
if (!document.querySelector("#window-styles")) {
  const styleSheet = document.createElement("style");
  styleSheet.id = "window-styles";
  styleSheet.textContent = windowStyles;
  document.head.appendChild(styleSheet);
}

// Export for use
if (typeof module !== "undefined" && module.exports) {
  module.exports = DesktopWindow;
}
