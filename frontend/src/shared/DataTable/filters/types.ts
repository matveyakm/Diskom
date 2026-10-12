// Состояние Excel-фильтров грида (попап по клику на шапку).
//
// Выводится из данных, не из кода: вид контрола решает пара
// (TableColumnConfig + FormField), здесь только форма состояния.
// Семантика (как в Excel): условие ∩ галочки внутри колонки,
// внутри галочек — ИЛИ, между колонками — И.

export type SortDir = "asc" | "desc";

export interface ColumnSort {
    key: string;
    dir: SortDir;
}

export type FilterOp =
    | "eq" | "ne"
    | "gt" | "gte" | "lt" | "lte"
    | "between" | "contains";

export interface ConditionFilter {
    op: FilterOp;
    /** Первое значение (для between — нижняя граница). */
    value: string;
    /** Верхняя граница, только для between. */
    value2?: string;
}

export interface ColumnFilterState {
    /** Условие-оператор (одно, без цепочек И/ИЛИ — осознанно). */
    cond?: ConditionFilter | null;
    /**
     * Набор галочек. null/undefined — фильтр неактивен (все);
     * пустой массив — всё снято (совпадений нет).
     */
    checked?: Array<string | number> | null;
}

export type TableColumnFilters = Record<string, ColumnFilterState>;

/**
 * Связка для шапки грида: сортировка одна на таблицу (бэкенд умеет
 * единственный ?sort=), фильтры — по колонкам.
 */
export interface HeaderFiltering {
    sort: ColumnSort | null;
    filters: TableColumnFilters;
    openKey: string | null;
    docName: string | undefined;
    onTogglePopup: (key: string | null) => void;
    onSort: (key: string, dir: SortDir | null) => void;
    onFilter: (key: string, state: ColumnFilterState | null) => void;
}
