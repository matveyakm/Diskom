import {useMemo, useState} from "react";
import {useQuery} from "@tanstack/react-query";
import styles from "./FilterEditor.module.scss";
import {TableColumnConfig} from "@/shared/DataTable/types";
import {FormField, resolveRefEndpoint} from "@/entities/document/types";
import {isDateField} from "@/shared/DataTable/dateUtils";
import {STATUS_OPTIONS} from "@/shared/DataTable/ui/statusMeta";
import {tableApi} from "@/shared/api/fetchTableData";
import type {DistinctResponse} from "@/shared/api/fetchTableData/types";
import {useReferenceOptions} from "@/shared/api/useReferenceOptions";
import type {ConditionFilter, FilterOp, SortDir} from "@/shared/DataTable/filters/types";

type ColumnKind = "status" | "ref" | "date" | "text";

const OP_LABELS: Record<FilterOp, string> = {
    eq: "равно",
    ne: "не равно",
    gt: "больше",
    gte: "больше или равно",
    lt: "меньше",
    lte: "меньше или равно",
    between: "между",
    contains: "содержит",
};

const FULL_OPS: FilterOp[] = ["eq", "ne", "gt", "gte", "lt", "lte", "between", "contains"];
const EQ_OPS: FilterOp[] = ["eq", "ne"];

/** Вид контрола выводится из данных: колонка грида (если поле в нём есть) + поле формы. */
const kindOfField = (column: TableColumnConfig | undefined, formField?: FormField): ColumnKind => {
    if (column?.type === "status") return "status";
    if (resolveRefEndpoint(column) ?? resolveRefEndpoint(formField)) return "ref";
    if (formField && isDateField(formField)) return "date";
    return "text";
};

interface FilterEditorProps {
    docName: string | undefined;
    fieldKey: string;
    column?: TableColumnConfig;
    formField?: FormField;
    sortDir: SortDir | null;
    cond: ConditionFilter | null;
    checked: Array<string | number> | null;
    /** Параметры фильтров соседних закладок: список сужается тем, что выбрано в них. */
    siblingParams: Record<string, string>;
    onSort: (dir: SortDir | null) => void;
    onCond: (cond: ConditionFilter | null) => void;
    onChecked: (values: Array<string | number> | null) => void;
    onResetField: () => void;
}

// Контролируемый редактор одного поля: всё состояние живёт у вызывающего
// (окно фильтров коммитит всё одной кнопкой), здесь только черновик поиска
// по галочкам. Монтируется заново на каждую закладку (key), поэтому
// локального состояния фильтров не держит.
export const FilterEditor = ({
    docName, fieldKey, column, formField, sortDir, cond, checked, siblingParams,
    onSort, onCond, onChecked, onResetField,
}: FilterEditorProps) => {
    const kind = kindOfField(column, formField);
    const ops = kind === "text" || kind === "date" ? FULL_OPS : EQ_OPS;
    // Выбранный оператор живёт локально: пустое условие в черновик не
    // пишется (иначе селект схлопывался бы обратно на "равно").
    const [op, setOp] = useState<FilterOp>(cond?.op ?? "eq");
    const [checkSearch, setCheckSearch] = useState("");

    // Значения галочек: статусы фиксированы, остальное — ?distinct=,
    // суженный фильтрами соседних закладок (свой столбец бэкенд вычитает сам).
    // Имена ссылок — из общего кеша ["reference", end_point].
    const endPoint = resolveRefEndpoint(column) ?? resolveRefEndpoint(formField);
    const {data: distinct, isLoading: distinctLoading} = useQuery({
        queryKey: ["distinct", docName, fieldKey, siblingParams],
        queryFn: () => tableApi(docName ?? "").getDistinct(fieldKey, siblingParams) as Promise<DistinctResponse>,
        enabled: !!docName && kind !== "status",
        staleTime: 60_000,
    });
    const {data: refOptions} = useReferenceOptions(kind === "ref" ? endPoint : undefined);

    const refNames = useMemo(() => {
        const map = new Map<string, string>();
        for (const o of refOptions ?? []) map.set(String(o.id), o.name);
        return map;
    }, [refOptions]);

    const allValues: Array<string | number> = useMemo(() => {
        if (kind === "status") return STATUS_OPTIONS.map(o => o.value);
        const vals = (distinct?.data ?? []).filter(v => v !== null) as Array<string | number>;
        return vals;
    }, [kind, distinct]);

    const labelOf = (v: string | number): string => {
        if (kind === "status") {
            return STATUS_OPTIONS.find(o => o.value === Number(v))?.label ?? String(v);
        }
        if (kind === "ref") {
            return refNames.get(String(v)) ?? String(v);
        }
        return String(v);
    };

    const visibleValues = useMemo(() => {
        const q = checkSearch.trim().toLowerCase();
        if (!q) return allValues;
        return allValues.filter(v =>
            labelOf(v).toLowerCase().includes(q) || String(v).toLowerCase().includes(q),
        );
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [allValues, checkSearch, refNames, kind]);

    const allChecked = checked !== null && checked.length === allValues.length && allValues.length > 0;

    const toggleValue = (v: string | number) => {
        if (allValues.length === 0) return;
        const cur = checked ?? allValues;
        const s = String(v);
        onChecked(
            cur.some(x => String(x) === s)
                ? cur.filter(x => String(x) !== s)
                : [...cur, v],
        );
    };

    const toggleSort = (dir: SortDir) => {
        onSort(sortDir === dir ? null : dir);
    };

    const setCondValue = (value: string) => {
        if (op === "between") {
            const v2 = cond?.value2 ?? "";
            if (value.trim() === "" && v2.trim() === "") onCond(null);
            else onCond({op, value: value.trim(), value2: v2.trim()});
        } else if (value.trim() === "") {
            onCond(null);
        } else {
            onCond({op, value: value.trim()});
        }
    };

    const setCondValue2 = (value2: string) => {
        const v = cond?.value ?? "";
        if (v.trim() === "" && value2.trim() === "") onCond(null);
        else onCond({op, value: v.trim(), value2: value2.trim()});
    };

    const setCondOp = (next: FilterOp) => {
        setOp(next);
        const v = cond?.value ?? "";
        const v2 = cond?.value2 ?? "";
        if (next === "between") {
            if (v.trim() === "" && v2.trim() === "") onCond(null);
            else onCond({op: next, value: v.trim(), value2: v2.trim()});
        } else if (v.trim() === "") {
            onCond(null);
        } else {
            onCond({op: next, value: v.trim()});
        }
    };

    const condOp: FilterOp = op;
    const condValue = cond?.value ?? "";
    const condValue2 = cond?.value2 ?? "";

    const renderValueInput = (value: string, setValue: (v: string) => void) => {
        if (kind === "ref") {
            return (
                <select value={value} onChange={(e) => setValue(e.target.value)} className={styles.input}>
                    <option value="">—</option>
                    {(refOptions ?? []).map(o => (
                        <option key={o.id} value={String(o.id)}>{o.name}</option>
                    ))}
                </select>
            );
        }
        if (kind === "status") {
            return (
                <select value={value} onChange={(e) => setValue(e.target.value)} className={styles.input}>
                    <option value="">—</option>
                    {STATUS_OPTIONS.map(o => (
                        <option key={o.value} value={String(o.value)}>{o.label}</option>
                    ))}
                </select>
            );
        }
        if (kind === "date") {
            return (
                <input type="date" value={value} onChange={(e) => setValue(e.target.value)} className={styles.input}/>
            );
        }
        return (
            <input type="text" value={value} onChange={(e) => setValue(e.target.value)} className={styles.input}
                   placeholder="Значение…"/>
        );
    };

    return (
        <div className={styles.editor}>
            <div className={styles.section}>
                <div className={styles.sectionTitle}>Сортировка</div>
                <div className={styles.sortRow}>
                    <button type="button"
                            className={`${styles.sortBtn} ${sortDir === "asc" ? styles.active : ""}`}
                            onClick={() => toggleSort("asc")}>
                        По возрастанию {sortDir === "asc" ? "▲" : ""}
                    </button>
                    <button type="button"
                            className={`${styles.sortBtn} ${sortDir === "desc" ? styles.active : ""}`}
                            onClick={() => toggleSort("desc")}>
                        По убыванию {sortDir === "desc" ? "▼" : ""}
                    </button>
                </div>
            </div>

            <div className={styles.section}>
                <div className={styles.sectionTitle}>Условие</div>
                <div className={styles.condRow}>
                    <select value={condOp} onChange={(e) => setCondOp(e.target.value as FilterOp)}
                            className={styles.input}>
                        {ops.map(op => (
                            <option key={op} value={op}>{OP_LABELS[op]}</option>
                        ))}
                    </select>
                    {renderValueInput(condValue, setCondValue)}
                </div>
                {condOp === "between" && (
                    <div className={styles.condRow}>
                        <span className={styles.betweenLabel}>и</span>
                        {renderValueInput(condValue2, setCondValue2)}
                    </div>
                )}
            </div>

            <div className={styles.section}>
                <div className={styles.sectionTitle}>Значения</div>
                <input type="text" value={checkSearch} onChange={(e) => setCheckSearch(e.target.value)}
                       className={styles.input} placeholder="Поиск по значениям…"/>
                <label className={styles.checkAll}>
                    <input type="checkbox"
                           ref={(el) => {
                               if (el) el.indeterminate = checked !== null && checked.length > 0 && !allChecked;
                           }}
                           checked={checked === null || allChecked}
                           onChange={() => onChecked(allChecked || checked === null ? [] : null)}/>
                    (Выбрать все)
                </label>
                <div className={styles.checkList}>
                    {distinctLoading && kind !== "status" ? (
                        <div className={styles.emptyChecks}>Загрузка значений…</div>
                    ) : (
                        <>
                            {visibleValues.map(v => {
                                const s = String(v);
                                const isOn = checked === null || checked.some(x => String(x) === s);
                                return (
                                    <label key={s} className={styles.checkItem}>
                                        <input type="checkbox" checked={isOn} onChange={() => toggleValue(v)}/>
                                        {labelOf(v)}
                                    </label>
                                );
                            })}
                            {visibleValues.length === 0 && (
                                <div className={styles.emptyChecks}>Нет значений</div>
                            )}
                        </>
                    )}
                </div>
                {distinct && distinct.truncated && (
                    <div className={styles.truncatedNote}>
                        Первые {distinct.data.length} из {distinct.total_distinct} — сузьте условием
                    </div>
                )}
            </div>

            <button type="button" onClick={onResetField} className={styles.resetFieldBtn}>
                Сбросить поле
            </button>
        </div>
    );
};
