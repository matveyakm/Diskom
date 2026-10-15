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
    filter?: string;
    /** New shape: ordered filter list (engine uses filter; first entry mirrors it). */
    filters?: string[];
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
