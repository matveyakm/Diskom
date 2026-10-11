// Описание последствий проведения документа — превью для confirm-диалога
// кнопок "Провести" / "Сохранить и провести".
//
// Философия как у движка: поведение — в данных. Функция не знает имён
// полей и конкретных сущностей, только роль шага в конфиге on_acceptance:
//   - GET скрывается всегда (чтение, данные не меняет),
//   - шаги в шапку (`document`) и в собственную строку вида скрываются
//     (внутренний переворот status, пользователю неинтересен),
//   - всё остальное описывается честным глаголом строго по method:
//     POST — создание, PUT — изменение, DELETE — удаление.
//     Содержимое шага (какие поля в какие) не интерпретируется осознанно:
//     это было бы хардкодом.
// Человеческое имя сущности подставляет вызывающий через resolveTitle
// (по menuData: table_title → name → сырой end_point, fail-open).

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

export type AcceptanceEffectVerb = "create" | "update" | "delete";

export interface AcceptanceEffect {
    verb: AcceptanceEffectVerb;
    endPoint: string;
    title: string;
    conditional: boolean;
}

const TECHNICAL_END_POINT = "document";

export const describeAcceptanceEffects = (
    steps: ProcessingStep[] | null | undefined,
    currentRef: string | null | undefined,
    resolveTitle: (endPoint: string) => string,
): AcceptanceEffect[] => {
    if (!steps?.length) return [];
    const out: AcceptanceEffect[] = [];
    for (const step of steps) {
        if (!step || typeof step.end_point !== "string" || step.end_point === "") continue;
        const method = (step.method || "get").toLowerCase();
        // Чтение и технические перевороты статуса — не последствия для пользователя.
        if (method === "get") continue;
        if (step.end_point === TECHNICAL_END_POINT) continue;
        if (currentRef && step.end_point === currentRef) continue;
        let verb: AcceptanceEffectVerb | null = null;
        if (method === "post") verb = "create";
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

export const formatAcceptanceEffect = (effect: AcceptanceEffect): string => {
    const head =
        effect.verb === "create" ? "Новая запись"
        : effect.verb === "update" ? "Изменение записи"
        : "Удаление";
    const scope = effect.verb === "delete" ? "из" : "в";
    const tail = effect.conditional ? " (условно)" : "";
    return `${head} ${scope} «${effect.title}»${tail}`;
};
