// WindowManager: facade (single responsibility: orchestrate one catalog
// window — config, data, pipeline, view). Parsing lives in configParser,
// step execution in PipelineEngine (transport injected), rendering in views.
class WindowManager {
  constructor(options = {}) {
    this.jsonColumns = options.table_columns;
    this.title = options.table_title;
    this.ref = options.ref;
    this.columns = null;
    this.itemFields = null;
    this.itemBlocks = null;
    this.tableData = null;
    this.itemTitle = options.item_title;
    this.jsonItemFields = options.item_fields;
    this.window = null;
    this.systemObjectId = options.id;
    this.systemObjectGroupId = Number(options.id_group) || null;

    this.processing = {
      onSaveNew: parseActionConfig(options.on_save_new),
      onSaveEdit: parseActionConfig(options.on_save_edit),
      onAcceptance: parseActionConfig(options.on_acceptance),
      onDelete: parseActionConfig(options.on_delete),
    };
    this.engine = new PipelineEngine(
      (endPoint, keys) =>
        new ApiService({ ref: "/api/" + endPoint, keys, useCache: false }),
    );
  }

  // ─── Defaults ─────────────────────────────────────────────────────────────
  // Применяет default-значения из itemFields к объекту item.
  // context — дополнительный источник для field-reference (например, после FK)
  //
  // Типы default:
  //   "now()"        → текущая дата/время ISO
  //   "id"           → значение поля "id" из context (field reference)
  //   1, 0, true     → литерал
  applyDefaults(item, context = null) {
    if (!this.itemFields?.length) return item;

    const src = context ?? item;
    const result = { ...item };

    this.itemFields.forEach(({ field, default: def }) => {
      if (def === undefined) return;

      // Не перезаписываем, если значение уже есть
      const current = result[field];
      if (current !== undefined && current !== null && current !== "") return;

      if (def === "now()") {
        // Текущая дата в формате YYYY-MM-DDTHH:mm (для input[type=datetime-local])
        result[field] = new Date().toISOString().slice(0, 16);
      } else if (typeof def === "string") {
        // Field reference: берём значение из src (item или context)
        result[field] = src[def] ?? null;
      } else {
        // Литерал: число, булево, null
        result[field] = def;
      }
    });

    return result;
  }

  // ─── Подготовка данных ────────────────────────────────────────────────────

  prepareColumns() {
    const parsed = parseColumnsConfig(this.jsonColumns, this.jsonItemFields);
    this.columns = parsed.columns;
    this.itemFields = parsed.itemFields;
    this.itemBlocks = parsed.itemBlocks;
  }

  async prepareData() {
    const api = new ApiService({ ref: "/api/" + this.ref, keys: {} });
    const res = await api.getResponse();
    this.tableData = res.data;
  }

  // ─── Движок обработки (делегировано PipelineEngine) ───────────────────────

  processAction(actionName, item) {
    const scoped = {
      ...item,
      id_system_object: item.id_system_object ?? this.systemObjectId ?? null,
    };
    return this.engine.processAction(actionName, scoped, this.processing[actionName], (obj, ctx) =>
      this.applyDefaults(obj, ctx),
    );
  }

  hashCode(obj) {
    const str = typeof obj === "string" ? obj : JSON.stringify(obj);
    let hash = 5381;
    for (let i = 0; i < str.length; i++) {
      hash = (hash * 33) ^ str.charCodeAt(i);
    }
    return (hash >>> 0).toString(16);
  }

  // ─── Публичные действия ───────────────────────────────────────────────────

  async saveNew(item) {
    item["id_transaction"] = this.hashCode(item);
    await this.processAction("onSaveNew", item);
    await this.refreshWindow();
  }

  async saveItem(item) {
    item["id_transaction"] = this.hashCode(item);
    await this.processAction("onSaveEdit", item);
    await this.refreshWindow();
  }

  async acceptItem(item) {
    await this.processAction("onAcceptance", item);
    await this.refreshWindow();
  }

  async deleteItem(item) {
    await this.processAction("onDelete", item);
    await this.refreshWindow();
  }

  // ─── Жизненный цикл окна ──────────────────────────────────────────────────

  async openWindow() {
    this.prepareColumns();
    await this.prepareData();
    this.window = new DesktopWindow({
      title: this.title,
      width: 800,
      height: 500,
      left: 150,
      top: 100,
      columns: this.columns,
      itemTitle: this.itemTitle,
      itemFields: this.itemFields,
      itemBlocks: this.itemBlocks,
      systemObjectId: this.systemObjectId,
      systemObjectGroupId: this.systemObjectGroupId,
      onSave: async (item) => {
        item["ddate"] = new Date(item["ddate"]).toJSON();
        item["id_system_object"] = this.systemObjectId;
        await this.saveNew(item);
      },
      onEdit: async (item) => {
        await this.saveItem(item);
      },
      onAccept: async (item) => {
        await this.acceptItem(item);
      },
      onDelete: async (item) => {
        await this.deleteItem(item);
      },
    });
    this.window.render();
    this.window.maximize();
    await this.refreshWindow();
  }

  async refreshWindow() {
    await this.prepareData();
    this.window.data = this.tableData;
    this.window.refreshData();
  }
}
