import { api } from "@/shared/api/AxiosClient";
import { FormField } from "@/entities/document/types";
import type { ProcessingStep } from "./effects";

export type { PipelineEffect, PipelineEffectVerb, ProcessingStep } from "./effects";
export { describePipelineEffects, formatPipelineEffect } from "./effects";

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

// Человекочитаемая причина падения шага для алертов: тело ответа бэка
// (там обычно точная ошибка вида Failed to ...), иначе message.
export const describeActionError = (err: unknown): string => {
    const data = (err as { response?: { data?: unknown } })?.response?.data;
    if (data !== undefined && data !== null) {
        const text = typeof data === "string" ? data : JSON.stringify(data);
        return text.length > 200 ? text.slice(0, 200) + "…" : text;
    }
    if (err instanceof Error && err.message) return err.message;
    return String(err ?? "неизвестная ошибка");
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
        // Readonly fields are backend-owned (e.g. auto-generated numbers):
        // defaults must never write into them, otherwise a numeric default
        // like "id" lands in a string column and breaks Go decoding.
        if (f.readonly) continue;

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

// firstRecordOf достаёт первую запись из ответа: одиночная строка как есть,
// пачка {data: [...]} — нулевой элемент, пустая пачка — undefined (merge и
// FK молча пропускают, вместо падения на конверте).
const firstRecordOf = (response: unknown): Record<string, unknown> | undefined => {
    const unwrapped = (response as Record<string, unknown>)?.data ?? response;
    const inner = Array.isArray(unwrapped)
        ? unwrapped[0]
        : Array.isArray((unwrapped as Record<string, unknown>)?.data)
          ? (unwrapped as { data: Record<string, unknown>[] }).data[0]
          : unwrapped;
    if (inner && typeof inner === "object") {
        return inner as Record<string, unknown>;
    }
    return undefined;
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

    // Строковый фильтр — same-name (?id=<ctx.id>).
    if (typeof step.filter === "string" && context[step.filter] !== undefined) {
        params[step.filter] = String(context[step.filter]);
    }

    // Объектный фильтр — переименование (?id=<ctx.id_employee>).
    // Отсутствующий ключ и нулевой FK пропускаются молча (идемпотентный
    // no-op чистки непроведённого — та же семантика, что у map-ключей).
    if (step.filter && typeof step.filter === "object" && !Array.isArray(step.filter)) {
        for (const [target, source] of Object.entries(step.filter as Record<string, unknown>)) {
            if (typeof source === "string") {
                if (!Object.prototype.hasOwnProperty.call(context, source)) continue;
                const v = context[source];
                if (v === null || v === undefined || v === "" || v === 0 || v === "0") continue;
                params[target] = String(v);
            } else if (source !== undefined && source !== null) {
                params[target] = String(source);
            }
        }
    }

    // DELETE: query строится ещё и из map (target←source из контекста;
    // отсутствующий ключ/нулевой FK пропускаются, литералы идут как есть).
    // PUT/POST map уезжает в тело (там его читает бэкенд).
    // Явные литералы query (не из контекста): например {"status": 2} —
    // static-фильтр, которого нет в форме (провести можно только проведённое,
    // удалить — только своё; движок literals не резолвит).
    if (step.query && typeof step.query === "object" && !Array.isArray(step.query)) {
        for (const [target, value] of Object.entries(step.query as Record<string, unknown>)) {
            if (value === undefined || value === null || value === "") continue;
            params[target] = String(value);
        }
    }
    if (method === "delete" && step.map) {
        const mappings = Array.isArray(step.map) ? step.map : [step.map];
        for (const mapping of mappings) {
            const [target, source] = Object.entries(mapping as Record<string, unknown>)[0];
            if (typeof source === "string") {
                if (!Object.prototype.hasOwnProperty.call(context, source)) continue;
                const v = context[source];
                if (v === null || v === undefined || v === "" || v === 0 || v === "0") continue;
                params[target] = String(v);
            } else if (source !== undefined && source !== null) {
                params[target] = String(source);
            }
        }
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
            // Первая запись ответа — и одиночная строка, и data[0] пачки
            // (без фильтра бэкенд отдаёт конверт {data, pagination}; сортировка
            // по умолчанию — id DESC, т.е. первой идёт самая новая запись).
            // merge:false — не вливать запись в контекст (нужен только FK:
            // иначе чужая строка затопчет поля формы для следующих шагов).
            const record = firstRecordOf(response);
            console.log(`[actionPipeline] GET record:`, JSON.stringify(record));
            if (record && typeof record === "object" && step.merge !== false) {
                Object.assign(context, record);
                console.log(`[actionPipeline] Context after GET:`, JSON.stringify(context));
            }
        } else if (method === "delete") {
            // Query — единственный рабочий путь: path-style бэкенд не
            // маршрутизирует (падает в заглушку / с тихим 200 — молчаливый
            // no-op вместо удаления). Пустой DELETE — честный 400 с бэка.
            console.log(`[actionPipeline] DELETE ${url}${queryString}`);
            await api.delete(`${url}${queryString}`);
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

    const responseData = firstRecordOf(response);
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
