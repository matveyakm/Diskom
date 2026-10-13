// Описание последствий пайплайна — превью для confirm-диалогов
// ("Провести", "Сохранить и провести", "Удалить").
//
// Поведение — в данных: функция не знает имён полей, сущностей и того,
// какие шаги "технические". Единственный источник скрытия — флаг
// "hide_in_preview": true на самом шаге (строго boolean; мусор вроде
// "yes" не скрывает — fail-closed на своём синтаксисе).
// Нет флага — шаг виден (fail-open: немаркированное лучше показать,
// чем потерять). Глагол — строго по method:
//   GET — прочитать, POST — создать, PUT — изменить, DELETE — удалить.
// Неизвестный method пропускается. На исполнение шагов флаг не влияет
// (движок его игнорирует — читает только известные поля).

export interface ProcessingStep {
    Step?: number;
    step?: number;
    method?: string;
    end_point: string;
    hide_in_preview?: boolean;
    map?: Record<string, unknown>[];
    fields?: Record<string, unknown>[];
    /**
     * Идентификация строки для GET/PUT/DELETE. Строка — same-name
     * (`"id"` → `?id=<ctx.id>`); объект — переименование
     * (`{"id": "id_employee"}` → `?id=<ctx.id_employee>`) для случаев,
     * когда нужного ключа в строке нет, а значение есть под другим именем.
     * Нерезолвящиеся ключи пропускаются (см. build*Query).
     */
    filter?: string | Record<string, string>;
    FK?: Record<string, string>[];
    conditions?: Record<string, unknown>[];
}

export type PipelineEffectVerb = "read" | "create" | "update" | "delete";

export interface PipelineEffect {
    verb: PipelineEffectVerb;
    endPoint: string;
    title: string;
    conditional: boolean;
}

// Единственный резолвер "имя → значение из контекста, иначе литерал"
// для тел POST/PUT и query DELETE. Главная мина (см. MEMORY): отсутствующий
// ключ возвращает буквальную строку-имя — лечится полнотой контекста.
export const resolveValue = (source: unknown, context: Record<string, unknown>): unknown => {
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

export const describePipelineEffects = (
    steps: ProcessingStep[] | null | undefined,
    resolveTitle: (endPoint: string) => string,
): PipelineEffect[] => {
    if (!steps?.length) return [];
    const out: PipelineEffect[] = [];
    for (const step of steps) {
        if (!step || typeof step.end_point !== "string" || step.end_point === "") continue;
        if (step.hide_in_preview === true) continue;
        const method = (step.method || "get").toLowerCase();
        let verb: PipelineEffectVerb | null = null;
        if (method === "get") verb = "read";
        else if (method === "post") verb = "create";
        else if (method === "put") verb = "update";
        else if (method === "delete") verb = "delete";
        else continue;
        const resolved = resolveTitle(step.end_point);
        out.push({
            verb,
            endPoint: step.end_point,
            title: resolved || step.end_point,
            conditional: Array.isArray(step.conditions) && step.conditions.length > 0,
        });
    }
    return out;
};

export const formatPipelineEffect = (effect: PipelineEffect): string => {
    const head =
        effect.verb === "read" ? "Прочитать запись"
        : effect.verb === "create" ? "Новая запись"
        : effect.verb === "update" ? "Изменение записи"
        : "Удаление";
    const scope = effect.verb === "delete" ? "из" : "в";
    const tail = effect.conditional ? " (условно)" : "";
    return `${head} ${scope} «${effect.title}»${tail}`;
};

// Чистое построение query для DELETE-шага (и только для него).
// Источник — map шага (target — ключ запроса, source — ключ контекста):
// в отличие от тел POST/PUT, отсутствующий ключ контекста здесь НЕ
// превращается в буквальную строку-имя, а пропускается — литерал вместо
// id в query давал бы 500 от Postgres или тихий no-op, а пустой результат
// бэкенд честно отвергнет 400 (защита от массового сноса).
// fields шага — литералы поверх (как в buildBody).
// Без map — resolveFilterEntries (см. ниже).
export const buildDeleteQuery = (
    step: ProcessingStep,
    context: Record<string, unknown>,
): string => {
    const params: Record<string, string> = {};

    const push = (key: string, value: unknown) => {
        if (value === undefined || value === null) return;
        params[key] = String(value);
    };

    if (hasStepMap(step)) {
        for (const [target, source] of mapEntries(step)) {
            if (typeof source === "string" && !hasOwnKey(context, source)) continue;
            push(target, resolveValue(source, context));
        }
    } else {
        Object.assign(params, resolveFilterEntries(step.filter, context));
    }

    if (Array.isArray(step.fields)) {
        for (const f of step.fields) {
            const [key, val] = Object.entries(f)[0];
            push(key, val);
        }
    } else if (step.fields && typeof step.fields === "object") {
        for (const [key, val] of Object.entries(step.fields)) {
            push(key, val);
        }
    }

    return toQueryString(params);
};

// Чистое построение query для PUT-шага. Только filter (map у PUT строит
// ТЕЛО, не query): строковая форма — same-name, объектная — переименование.
// Нерезолвящиеся ключи пропускаются.
export const buildPutQuery = (
    step: ProcessingStep,
    context: Record<string, unknown>,
): string => toQueryString(resolveFilterEntries(step.filter, context));

// filter → пары query. Пустой результат — нормально: вызывающий решает
// (DELETE/PUT без query бэкенд отвергнет 400; шаг с объектным filter без
// резолва движок пропускает как идемпотентный no-op чистки).
export const resolveFilterEntries = (
    filter: ProcessingStep["filter"],
    context: Record<string, unknown>,
): Record<string, string> => {
    const params: Record<string, string> = {};
    if (!filter) return params;
    if (typeof filter === "string") {
        if (filter === "id") {
            if (context.id !== undefined && context.id !== null) params.id = String(context.id);
        } else if (context[filter] !== undefined && context[filter] !== null) {
            params[filter] = String(context[filter]);
        }
        return params;
    }
    for (const [queryKey, contextKey] of Object.entries(filter)) {
        if (context[contextKey] !== undefined && context[contextKey] !== null) {
            params[queryKey] = String(context[contextKey]);
        }
    }
    return params;
};

const hasOwnKey = (context: Record<string, unknown>, key: string): boolean =>
    Object.prototype.hasOwnProperty.call(context, key);

const hasStepMap = (step: ProcessingStep): boolean =>
    Array.isArray(step.map) || (!!step.map && typeof step.map === "object");

const mapEntries = (step: ProcessingStep): Array<[string, unknown]> => {
    if (Array.isArray(step.map)) {
        return step.map.map(m => {
            const [target, source] = Object.entries(m)[0];
            return [target, source] as [string, unknown];
        });
    }
    if (step.map && typeof step.map === "object") {
        return Object.entries(step.map);
    }
    return [];
};

const toQueryString = (params: Record<string, string>): string => {
    const keys = Object.keys(params);
    return keys.length ? "?" + new URLSearchParams(params).toString() : "";
};
