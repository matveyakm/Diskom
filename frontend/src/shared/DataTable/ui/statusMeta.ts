// Человеческие имена статусов — единый источник для иконки,
// галочек и селектов Excel-фильтра (отдельный модуль без JSX,
// чтобы не нарушать react-refresh/only-export-components).
export const STATUS_LABELS: Record<number, string> = {
    1: "Сохранён",
    2: "Проведён",
    0: "Удалён",
};

/** Фиксированный набор статусов для фильтров. */
export const STATUS_OPTIONS: Array<{ value: number; label: string }> = [1, 2, 0].map(
    (value) => ({value, label: STATUS_LABELS[value]}),
);
