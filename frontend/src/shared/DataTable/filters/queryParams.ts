// Маппинг состояния Excel-фильтров в query-параметры shared-бэкенда.
// Чистая функция без импортов (проверяется изолированно через tsc+node).
// Пустые значения условия — не фильтр (шаг пропускается); пустой набор
// галочек — осознанно "совпадений нет" (бэкенд вернёт []).
import type {ColumnSort, TableColumnFilters} from "./types";

const condParam = (key: string, op: string, value: string): [string, string] => {
    switch (op) {
        case "ne": return [`${key}__ne`, value];
        case "gt":
        case "gte":
        case "lt":
        case "lte":
            return [`${key}__${op}`, value];
        case "contains": return [`${key}__like`, value];
        case "eq":
        default:
            return [key, value];
    }
};

export interface ListQueryInput {
    sort?: ColumnSort | null;
    filters?: TableColumnFilters | null;
    /** Глобальный поиск (подстрока). */
    q?: string;
    /** Колонки глобального поиска (только локальные: имена *_name живут в других сервисах). */
    qFields?: string[];
}

export const buildListParams = (input: ListQueryInput): Record<string, string> => {
    const out: Record<string, string> = {};

    if (input.sort) {
        out.sort = input.sort.key;
        out.order = input.sort.dir;
    }

    const filters = input.filters ?? {};
    for (const [key, state] of Object.entries(filters)) {
        if (!state) continue;
        const cond = state.cond;
        if (cond) {
            if (cond.op === "between") {
                const lo = (cond.value ?? "").trim();
                const hi = (cond.value2 ?? "").trim();
                if (lo !== "") out[`${key}__gte`] = lo;
                if (hi !== "") out[`${key}__lte`] = hi;
            } else {
                const v = (cond.value ?? "").trim();
                if (v !== "") {
                    const [k, val] = condParam(key, cond.op, v);
                    out[k] = val;
                }
            }
        }
        if (state.checked != null) {
            out[`${key}__in`] = state.checked.map(String).join(",");
        }
    }

    const q = (input.q ?? "").trim();
    if (q !== "" && (input.qFields ?? []).length > 0) {
        out.q = q;
        out.q_fields = input.qFields!.join(",");
    }

    return out;
};
