/* Field-editor factory (open/closed principle: new editor kinds are added
 * by registering a builder, without touching the form modal code). */
class FieldEditorFactory {
  constructor() {
    this.builders = new Map();
    this.register("textarea", (col, group) => this.buildTextarea(col, group));
    this.register("reference", (col, group) => this.buildReference(col, group));
    this.register("date", (col, group) => this.buildDate(col, group));
    this.register("readonly", (col, group) => this.buildReadonly(col, group));
    this.register("default", (col, group) => this.buildInput(col, group));
  }

  register(kind, builder) {
    this.builders.set(kind, builder);
  }

  kindOf(col) {
    if (col.readonly) return "readonly";
    if (col.type === "textarea") return "textarea";
    if (isRefField(col)) return "reference";
    if (col.field === "ddate") return "date";
    return "default";
  }

  // Returns { group, refConfig }: refConfig is non-null only for
  // reference editors and carries { field, endPoint, multiple, container }.
  build(col) {
    const group = document.createElement("div");
    group.className = "form-group";
    group.id = `group-${col.field}`;

    const label = document.createElement("label");
    label.textContent = col.label;
    label.htmlFor = `field-${col.field}`;
    group.appendChild(label);

    const builder = this.builders.get(this.kindOf(col)) || this.builders.get("default");
    return builder(col, group);
  }

  buildTextarea(col, group) {
    const input = document.createElement("textarea");
    input.rows = 3;
    input.id = `field-${col.field}`;
    input.name = col.field;
    input.className = "form-input";
    group.appendChild(input);
    return { group, refConfig: null };
  }

  buildReference(col, group) {
    const selectContainer = document.createElement("div");
    selectContainer.id = `select-container-${col.field}`;
    selectContainer.className = "enhanced-select-wrapper";
    group.appendChild(selectContainer);
    return {
      group,
      refConfig: {
        container: selectContainer,
        endPoint: refOf(col),
        field: col.field,
        multiple: col.multiple || false,
      },
    };
  }

  buildDate(col, group) {
    const { group: filled } = this.buildInput(col, group);
    const input = filled.querySelector(`#field-${col.field}`);
    if (input) input.type = "datetime-local";
    return { group: filled, refConfig: null };
  }

  // Backend-owned value (auto-generated document number): render as text,
  // keep the value in a hidden input so the form still submits it unchanged.
  // Never marked required — an empty value is valid and filled by the server.
  buildReadonly(col, group) {
    const display = document.createElement("div");
    display.className = "form-readonly";
    display.id = `readonly-${col.field}`;
    group.appendChild(display);

    const input = document.createElement("input");
    input.type = "hidden";
    input.id = `field-${col.field}`;
    input.name = col.field;
    input.className = "form-input";
    group.appendChild(input);
    return { group, refConfig: null };
  }

  buildInput(col, group) {
    const input = document.createElement("input");
    input.type = col.type || "text";
    input.id = `field-${col.field}`;
    input.name = col.field;
    input.className = "form-input";
    input.placeholder = col.placeholder || `Введите ${col.label.toLowerCase()}`;
    input.required = col.required || false;
    if (col.pattern) input.pattern = col.pattern;
    group.appendChild(input);
    return { group, refConfig: null };
  }
}
