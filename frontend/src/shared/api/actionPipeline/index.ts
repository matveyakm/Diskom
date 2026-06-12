import { api } from "@/shared/api/AxiosClient";
import { FormField } from "@/entities/document/types";

export interface ProcessingStep {
    Step?: number;
    step?: number;
    method?: string;
    end_point: string;
    map?: Record<string, unknown>[];
    fields?: Record<string, unknown>[];
    filter?: string;
    FK?: Record<string, string>[];
    conditions?: Record<string, unknown>[];
}

export interface ActionPipelineConfig {
    action: string;
    processing: ProcessingStep[];
}

export const parseActionConfig = (raw: string | Record<string, unknown> | null | undefined): ProcessingStep[] | null => {
    if (!raw || raw === "") return null;
    try {
        const config: ActionPipelineConfig = typeof raw === "string" ? JSON.parse(raw) : (raw as unknown as ActionPipelineConfig);
        return Array.isArray(config.processing) ? config.processing : null;
    } catch {
        return null;
    }
};

export const applyDefaults = (
    item: Record<string, unknown>,
    fields: FormField[],
    context: Record<string, unknown> | null = null,
): Record<string, unknown> => {
    if (!fields?.length) return item;
    const src = context ?? item;
    const result = { ...item };

    for (const f of fields) {
        const def = f.default;
        if (def === undefined) continue;

        const current = result[f.field];
        if (current !== undefined && current !== null && current !== "") continue;

        if (def === "now()") {
            result[f.field] = new Date().toISOString().slice(0, 16);
        } else if (typeof def === "string") {
            result[f.field] = src[def] ?? null;
        } else {
            result[f.field] = def;
        }
    }

    return result;
};

const resolveValue = (source: unknown, context: Record<string, unknown>): unknown => {
    if (typeof source === "string") {
        return Object.prototype.hasOwnProperty.call(context, source)
            ? (context[source] ?? null)
            : source;
    }
    if (source && typeof source === "object" && !Array.isArray(source)) {
        const rec = source as Record<string, unknown>;
        if (rec.CONCAT && Array.isArray(rec.CONCAT)) {
            return rec.CONCAT.map((part: unknown) =>
                typeof part === "string" && Object.prototype.hasOwnProperty.call(context, part)
                    ? String(context[part] ?? "")
                    : String(part ?? ""),
            ).join("");
        }
    }
    return source;
};

const checkConditions = (
    conditions: Record<string, unknown>[] | undefined,
    context: Record<string, unknown>,
): boolean => {
    if (!conditions?.length) return true;
    return conditions.every((cond) => {
        const [key, expected] = Object.entries(cond)[0];
        // eslint-disable-next-line eqeqeq
        return context[key] == expected;
    });
};

const buildBody = (
    step: ProcessingStep,
    context: Record<string, unknown>,
): Record<string, unknown> => {
    const body: Record<string, unknown> = {};

    if (Array.isArray(step.map)) {
        for (const mapping of step.map) {
            const [target, source] = Object.entries(mapping)[0];
            body[target] = resolveValue(source, context);
        }
    } else if (step.map && typeof step.map === "object") {
        for (const [target, source] of Object.entries(step.map)) {
            body[target] = resolveValue(source, context);
        }
    } else {
        Object.assign(body, context);
    }

    if (Array.isArray(step.fields)) {
        for (const f of step.fields) {
            const [key, val] = Object.entries(f)[0];
            body[key] = val;
        }
    } else if (step.fields && typeof step.fields === "object") {
        Object.assign(body, step.fields);
    }

    return body;
};

const executeStep = async (
    step: ProcessingStep,
    context: Record<string, unknown>,
): Promise<void> => {
    const method = (step.method || "get").toLowerCase();

    if (!checkConditions(step.conditions, context)) {
        console.log(`[actionPipeline] Step skipped: conditions not met for ${step.end_point}`);
        return;
    }

    const url = `/api/${step.end_point}`;
    const params: Record<string, string> = {};

    if (step.filter && context[step.filter] !== undefined) {
        params[step.filter] = String(context[step.filter]);
    }

    const queryString = Object.keys(params).length
        ? "?" + new URLSearchParams(params).toString()
        : "";

    let response;
    console.log(`[actionPipeline] Step: ${method.toUpperCase()} ${url}${queryString}`);

    try {
        if (method === "get") {
            const fullUrl = `${url}${queryString}`;
            console.log(`[actionPipeline] GET ${fullUrl}`);
            response = await api.get<unknown>(fullUrl);
            console.log(`[actionPipeline] GET response:`, JSON.stringify(response));
            const unwrapped = (response as Record<string, unknown>)?.data ?? response;
            console.log(`[actionPipeline] GET unwrapped:`, JSON.stringify(unwrapped));
            if (unwrapped) {
                const record = Array.isArray(unwrapped) ? (unwrapped as unknown[])[0] : unwrapped;
                console.log(`[actionPipeline] GET record:`, JSON.stringify(record));
                if (record && typeof record === "object") {
                    Object.assign(context, record as Record<string, unknown>);
                    console.log(`[actionPipeline] Context after GET:`, JSON.stringify(context));
                }
            }
        } else if (method === "delete") {
            const filterId = step.filter === "id" ? context.id : undefined;
            const delUrl = filterId != null ? `${url}/${encodeURIComponent(String(filterId))}` : url;
            console.log(`[actionPipeline] DELETE ${delUrl}`);
            await api.delete(delUrl);
            console.log(`[actionPipeline] DELETE success`);
        } else {
            const body = buildBody(step, context);
            console.log(`[actionPipeline] ${method.toUpperCase()} ${url}${queryString} body:`, JSON.stringify(body));
            if (method === "post") response = await api.post(`${url}${queryString}`, body);
            if (method === "put") response = await api.put(`${url}${queryString}`, body);
            console.log(`[actionPipeline] ${method.toUpperCase()} response:`, JSON.stringify(response));
        }
    } catch (err) {
        console.error(`[actionPipeline] Error in step ${step.end_point}:`, err);
        throw err;
    }

    const responseData = (response as Record<string, unknown>)?.data ?? response;
    if (step.FK && Array.isArray(step.FK) && responseData) {
        console.log(`[actionPipeline] FK mapping from response:`, JSON.stringify(responseData));
        for (const fkMap of step.FK) {
            const [ctxField, respField] = Object.entries(fkMap)[0];
            const respRecord = responseData as Record<string, unknown>;
            if (respRecord[respField] !== undefined) {
                (context as Record<string, unknown>)[ctxField] = respRecord[respField];
                console.log(`[actionPipeline] FK: context.${ctxField} = ${respRecord[respField]}`);
            }
        }
    }
};

export const processAction = async (
    actionName: string,
    item: Record<string, unknown>,
    pipelineConfigStr: string | null | undefined,
    fields: FormField[],
    systemObjectId?: number | null,
): Promise<Record<string, unknown>> => {
    console.log(`[actionPipeline] ===== ${actionName} =====`);
    console.log(`[actionPipeline] item:`, JSON.stringify(item));
    console.log(`[actionPipeline] systemObjectId:`, systemObjectId);
    console.log(`[actionPipeline] config:`, pipelineConfigStr);

    const steps = parseActionConfig(pipelineConfigStr);
    if (!steps?.length) {
        console.log(`[actionPipeline] No steps, skipping pipeline`);
        return item;
    }

    console.log(`[actionPipeline] Parsed steps:`, JSON.stringify(steps));

    const context: Record<string, unknown> = {
        ...applyDefaults(item, fields),
        id_system_object: systemObjectId ?? item.id_system_object ?? null,
    };

    console.log(`[actionPipeline] Initial context:`, JSON.stringify(context));

    const sorted = [...steps].sort(
        (a, b) => (a.Step ?? a.step ?? 0) - (b.Step ?? b.step ?? 0),
    );

    for (const step of sorted) {
        console.log(`[actionPipeline] Executing step ${step.Step ?? step.step}: ${step.end_point}`);
        await executeStep(step, context);
        const updated = applyDefaults(context as Record<string, unknown>, fields, context);
        Object.assign(context, updated);
        console.log(`[actionPipeline] Context after step:`, JSON.stringify(context));
    }

    console.log(`[actionPipeline] ===== ${actionName} done =====`);
    return context;
};
