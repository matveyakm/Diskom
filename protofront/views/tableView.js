/* Table grid view (single responsibility: render rows + selection state.
 * Reference-name loading is injected (dependency inversion); the default
 * loader preserves the original inline-fetch behavior. */
class TableView {
  constructor(columns, { onEditItem = null, loadRefName = null } = {}) {
    this.columns = columns || [];
    this.onEditItem = onEditItem;
    this.loadRefName =
      loadRefName ||
      (async (col, value) => {
        const response = await fetch(`/api/${refOf(col)}/${value}`);
        const data = await response.json();
        return data.name || data.label || "-";
      });
  }

  createTable() {
    const table = document.createElement("table");
    table.className = "data-table";

    const thead = document.createElement("thead");
    const headerRow = document.createElement("tr");

    const selectTh = document.createElement("th");
    selectTh.innerHTML = '<input type="checkbox" class="select-all">';
    selectTh.style.width = "40px";
    headerRow.appendChild(selectTh);

    this.columns.forEach((col) => {
      const th = document.createElement("th");
      th.textContent = col.label;
      if (col.width) th.style.width = col.width;
      headerRow.appendChild(th);
    });

    thead.appendChild(headerRow);
    table.appendChild(thead);

    const tbody = document.createElement("tbody");
    table.appendChild(tbody);
    return table;
  }

  async renderRows(tbody, data) {
    tbody.innerHTML = "";

    if (!data || data.length === 0) {
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

    for (const item of data) {
      const row = document.createElement("tr");
      row.dataset.id = item.id;

      const selectCell = document.createElement("td");
      const checkbox = document.createElement("input");
      checkbox.type = "checkbox";
      checkbox.className = "row-select";
      checkbox.dataset.id = item.id;
      selectCell.appendChild(checkbox);
      row.appendChild(selectCell);

      for (const col of this.columns) {
        const cell = document.createElement("td");

        if (col.field === "status") {
          cell.classList.add("status-cell");
          cell.setAttribute("data-value", item[col.field]);
          cell.textContent = "";
        } else if (refOf(col)) {
          try {
            cell.textContent = await this.loadRefName(col, item[col.field]);
          } catch (error) {
            cell.textContent = "-";
          }
        } else {
          cell.textContent = item[col.field] || "-";
        }

        row.appendChild(cell);
      }

      if (this.onEditItem) {
        row.addEventListener("dblclick", () => this.onEditItem(item));
      }

      tbody.appendChild(row);
    }
  }

  getSelectedIds(root) {
    const checkboxes = root.querySelectorAll(".row-select:checked");
    return Array.from(checkboxes).map((cb) => {
      const id = cb.dataset.id;
      return isNaN(id) ? id : parseInt(id);
    });
  }

  setAllSelected(root, checked) {
    root.querySelectorAll(".row-select").forEach((cb) => (cb.checked = checked));
  }
}
