/* Action-pipeline engine (single responsibility: execute processing steps).
 * Transport is injected via apiFactory(endPoint, keys) (dependency inversion),
 * so the engine is testable without network and unaware of ApiService. */
class PipelineEngine {
  constructor(apiFactory, { log = console } = {}) {
    if (typeof apiFactory !== "function") {
      throw new Error("PipelineEngine: apiFactory is required");
    }
    this.apiFactory = apiFactory;
    this.log = log;
  }

  async processAction(actionName, item, steps, defaultsApplier) {
    if (!steps?.length) {
      console.warn(`PipelineEngine: нет конфига для "${actionName}"`);
      return item;
    }
    const context = {
      ...defaultsApplier(item),
      id_system_object: item.id_system_object ?? null,
    };
    const sorted = [...steps].sort(
      (a, b) => (a.Step ?? a.step ?? 0) - (b.Step ?? b.step ?? 0),
    );
    for (const step of sorted) {
      const num = step.Step ?? step.step;
      this.log.log(
        `[${actionName}] Step ${num}: ${step.method?.toUpperCase()} /api/${step.end_point}`,
      );
      try {
        await this.executeStep(step, context);
        // Re-apply defaults: field-reference defaults (e.g. "default": "id")
        // resolve only AFTER an FK returns an id from a response.
        Object.assign(context, defaultsApplier(context, context));
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
      console.log("Step skipped: conditions not met");
      return;
    }

    const keys = {};
    if (step.filter && context[step.filter] !== undefined) {
      keys[step.filter] = context[step.filter];
    }
    const api = this.apiFactory(step.end_point, keys);

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
}
