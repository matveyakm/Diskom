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
