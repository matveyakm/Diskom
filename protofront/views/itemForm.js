// ItemForm: add/edit modal lifecycle + data collection.
// Field DOM is delegated to FieldEditorFactory (open/closed);
// this class owns select-component registry and save flow.
class ItemForm {
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

    // Create form fields via the editor factory (new kinds register
    // there instead of extending this if/else chain).
    const factory = new FieldEditorFactory();
    this.fields.forEach((col) => {
      if (col.field !== "id" && !col.hidden) {
        const { group, refConfig } = factory.build(col);
        if (refConfig) {
          this.selectInstances[col.field] = refConfig;
        }
        modalBody.appendChild(group);
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
        if (isRefField(col) && this.selectInstances[col.field]) {
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
          isRefField(col) &&
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
      const result = await this.onAccept(acceptedItem);
      return result;
    }
    return null;
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
