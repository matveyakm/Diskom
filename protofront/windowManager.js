class WindowManager {
  constructor(options = {}) {
    this.jsonColumns = options.table_columns;
    this.title = options.table_title;
    this.ref = options.ref;
    this.columns = null;
    this.itemFields = null;
    this.tableData = null;
    this.itemTitle = options.item_title;
    this.jsonItemFields = options.item_fields;
    this.window = null;
    this.systemObjectId = options.id;
    this.systemObjectGroupId = Number(options.id_group) || null;

    this.processing = {
      onSaveNew: this.parseActionConfig(options.on_save_new),
      onSaveEdit: this.parseActionConfig(options.on_save_edit),
      onAcceptance: this.parseActionConfig(options.on_acceptance),
      onDelete: this.parseActionConfig(options.on_delete),
    };
  }

  parseActionConfig(raw) {
    if (!raw || raw === "") return null;
    try {
      const config = typeof raw === "string" ? JSON.parse(raw) : raw;
      return Array.isArray(config.processing) ? config.processing : null;
    } catch (e) {
      console.error("WindowManager: ошибка парсинга конфига:", e);
      return null;
    }
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
    try {
      this.columns = JSON.parse(this.jsonColumns).columns;
      const rawFields = this.jsonItemFields;
      this.itemFields = rawFields === "" ? [] : JSON.parse(rawFields).fields;
    } catch (e) {
      console.error("prepareColumns error:", e);
      this.columns = [];
      this.itemFields = [];
    }
  }

  async prepareData() {
    const api = new ApiService({ ref: "/api/" + this.ref, keys: {} });
    const res = await api.getResponse();
    this.tableData = res.data;
  }

  // ─── Движок обработки ─────────────────────────────────────────────────────

  async processAction(actionName, item) {
    const steps = this.processing[actionName];
    if (!steps?.length) {
      console.warn(`WindowManager: нет конфига для "${actionName}"`);
      return item;
    }

    // Применяем простые defaults (now(), литералы) до начала шагов
    const context = {
      ...this.applyDefaults(item),
      id_system_object: this.systemObjectId,
    };

    const sorted = [...steps].sort(
      (a, b) => (a.Step ?? a.step ?? 0) - (b.Step ?? b.step ?? 0),
    );

    for (const step of sorted) {
      const num = step.Step ?? step.step;
      console.log(
        `[${actionName}] Step ${num}: ${step.method?.toUpperCase()} /api/${step.end_point}`,
      );
      try {
        await this.executeStep(step, context);

        // После каждого шага повторно применяем defaults:
        // это нужно для field-reference defaults (например, "default": "id")
        // которые resolveются только ПОСЛЕ того как FK вернул id из ответа
        const updated = this.applyDefaults(context, context);
        Object.assign(context, updated);
      } catch (err) {
        console.error(`[${actionName}] Step ${num} failed:`, err);
        throw err;
      }
    }

    return context;
  }

  async executeStep(step, context) {
    const method = (step.method || "get").toLowerCase();

    if (!this.checkConditions(step.conditions, context)) {
      console.log(`Step skipped: conditions not met`);
      return;
    }

    const keys = {};
    if (step.filter && context[step.filter] !== undefined) {
      keys[step.filter] = context[step.filter];
    }

    const api = new ApiService({
      ref: "/api/" + step.end_point,
      keys,
      useCache: false,
    });

    let response;

    if (method === "get") {
      response = await api.getResponse();
      const data = response?.data ?? response;
      if (data) {
        const record = Array.isArray(data) ? data[0] : data;
        if (record) Object.assign(context, record);
      }
    } else if (method === "delete") {
      const filterId = step.filter === "id" ? context.id : null;
      response = await api.deleteResponse(filterId);
    } else {
      const body = this.buildBody(step, context);
      if (method === "post") response = await api.postResponse(body);
      if (method === "put") response = await api.putResponse(body);
    }

    const responseData = response?.data ?? response;
    if (step.FK && Array.isArray(step.FK) && responseData) {
      step.FK.forEach((fkMap) => {
        const [ctxField, respField] = Object.entries(fkMap)[0];
        if (responseData[respField] !== undefined) {
          context[ctxField] = responseData[respField];
          console.log(`FK: context.${ctxField} = ${responseData[respField]}`);
        }
      });
    }
  }

  buildBody(step, context) {
    const body = {};

    if (Array.isArray(step.map)) {
      step.map.forEach((mapping) => {
        const [target, source] = Object.entries(mapping)[0];
        body[target] = this.resolveValue(source, context);
      });
    } else if (step.map && typeof step.map === "object") {
      Object.entries(step.map).forEach(([target, source]) => {
        body[target] = this.resolveValue(source, context);
      });
    } else {
      Object.assign(body, context);
    }

    if (Array.isArray(step.fields)) {
      step.fields.forEach((f) => {
        const [key, val] = Object.entries(f)[0];
        body[key] = val;
      });
    } else if (step.fields && typeof step.fields === "object") {
      Object.assign(body, step.fields);
    }

    return body;
  }

  resolveValue(source, context) {
    if (typeof source === "string") {
      return Object.prototype.hasOwnProperty.call(context, source)
        ? (context[source] ?? null)
        : source;
    }
    if (source && typeof source === "object" && source.CONCAT) {
      return source.CONCAT.map((part) =>
        typeof part === "string" &&
        Object.prototype.hasOwnProperty.call(context, part)
          ? (context[part] ?? "")
          : String(part ?? ""),
      ).join("");
    }
    return source;
  }

  checkConditions(conditions, context) {
    if (!conditions?.length) return true;
    return conditions.every((cond) => {
      const [key, expected] = Object.entries(cond)[0];
      // eslint-disable-next-line eqeqeq
      return context[key] == expected;
    });
  }

  // ─── Хэш ─────────────────────────────────────────────────────────────────

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

  // ─── Жизненный цикл окна ─────────────────────────────────────────────────

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
