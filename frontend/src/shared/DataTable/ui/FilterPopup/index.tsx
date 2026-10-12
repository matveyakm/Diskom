import {useEffect, useMemo, useRef, useState} from "react";
import {useQuery} from "@tanstack/react-query";
import tableStyles from "@/shared/DataTable/DataTable.module.scss";
import styles from "./FilterPopup.module.scss";
import {TableColumnConfig} from "@/shared/DataTable/types";
import {FormField} from "@/entities/document/types";
import {isDateField} from "@/shared/DataTable/dateUtils";
import {STATUS_OPTIONS} from "@/shared/DataTable/ui/statusMeta";
import {tableApi} from "@/shared/api/fetchTableData";
import type {DistinctResponse} from "@/shared/api/fetchTableData/types";
import {useReferenceOptions} from "@/shared/api/useReferenceOptions";
import type {ColumnFilterState, FilterOp, SortDir} from "@/shared/DataTable/filters/types";

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

interface FilterPopupProps {
    docName: string | undefined;
    column: TableColumnConfig;
    formField?: FormField;
    sortDir: SortDir | null;
    filter: ColumnFilterState | null;
    onSort: (dir: SortDir | null) => void;
    onApplyFilter: (state: ColumnFilterState | null) => void;
    onClose: () => void;
}

interface HeaderCellProps {
    column: TableColumnConfig;
    formField?: FormField;
    docName: string | undefined;
    sortDir: SortDir | null;
    filter: ColumnFilterState | null;
    isOpen: boolean;
    onToggle: () => void;
    onClose: () => void;
    onSort: (dir: SortDir | null) => void;
    onApplyFilter: (state: ColumnFilterState | null) => void;
}

// Ячейка шапки: кнопка (Excel) + якорь попапа. Якорь display:contents,
// чтобы раскладка шапки не менялась; попап абсолютно позиционирован
// относительно .headerContent. Клик вне закрывает через общий хук.
export const HeaderCell = ({
    column, formField, docName, sortDir, filter, isOpen, onToggle, onClose, onSort, onApplyFilter,
}: HeaderCellProps) => {
    const anchorRef = useRef<HTMLDivElement>(null);
    // Клик вне якоря закрывает попап. Доступ к ref — только в эффекте
    // (не в рендере), как уже сделано в SelectField модалки.
    useEffect(() => {
        if (!isOpen) return;
        const el = anchorRef.current;
        if (!el) return;
        const handler = (e: MouseEvent) => {
            if (!el.contains(e.target as Node)) onClose();
        };
        document.addEventListener("mousedown", handler);
        return () => document.removeEventListener("mousedown", handler);
    }, [isOpen, onClose]);
    const hasFilter = !!filter && (filter.cond != null || filter.checked != null);
    return (
        <div ref={anchorRef} className={tableStyles.headerPopupAnchor}>
            <button type="button" className={tableStyles.headerFilterBtn} onClick={onToggle}
                    title="Сортировка и фильтр">
                <span className={tableStyles.headerFilterLabel}>{column.label || column.key}</span>
                {sortDir && <span>{sortDir === "asc" ? " ▲" : " ▼"}</span>}
                {hasFilter && <span className={tableStyles.filterMark}> ●</span>}
            </button>
            {isOpen && (
                <FilterPopup
                    docName={docName}
                    column={column}
                    formField={formField}
                    sortDir={sortDir}
                    filter={filter}
                    onSort={onSort}
                    onApplyFilter={onApplyFilter}
                    onClose={onClose}
                />
            )}
        </div>
    );
};

const kindOf = (column: TableColumnConfig, formField?: FormField): ColumnKind => {
    if (column.type === "status") return "status";
    if (column.end_point) return "ref";
    if (formField && isDateField(formField)) return "date";
    return "text";
};

export const FilterPopup = ({
    docName, column, formField, sortDir, filter, onSort, onApplyFilter, onClose,
}: FilterPopupProps) => {
    const kind = kindOf(column, formField);
    const ops = kind === "text" || kind === "date" ? FULL_OPS : EQ_OPS;

    const [condOp, setCondOp] = useState<FilterOp>(filter?.cond?.op ?? "eq");
    const [condValue, setCondValue] = useState(filter?.cond?.value ?? "");
    const [condValue2, setCondValue2] = useState(filter?.cond?.value2 ?? "");
    const [checked, setChecked] = useState<Array<string | number> | null>(filter?.checked ?? null);
    const [checkSearch, setCheckSearch] = useState("");

    // Значения галочек: статусы фиксированы, остальное — ?distinct=.
    // Имена ссылок — из общего кеша ["reference", end_point].
    const {data: distinct} = useQuery({
        queryKey: ["distinct", docName, column.key],
        queryFn: () => tableApi(docName ?? "").getDistinct(column.key) as Promise<DistinctResponse>,
        enabled: !!docName && kind !== "status",
        staleTime: 60_000,
    });
    const {data: refOptions} = useReferenceOptions(kind === "ref" ? column.end_point : undefined);

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
    const someChecked = checked !== null && checked.length > 0 && !allChecked;

    const toggleValue = (v: string | number) => {
        const cur = checked ?? allValues;
        const s = String(v);
        setChecked(
            cur.some(x => String(x) === s)
                ? cur.filter(x => String(x) !== s)
                : [...cur, v],
        );
    };

    const toggleSort = (dir: SortDir) => {
        onSort(sortDir === dir ? null : dir);
    };

    const handleApply = () => {
        const hasCond = condOp === "between"
            ? condValue.trim() !== "" || condValue2.trim() !== ""
            : condValue.trim() !== "";
        const cond = hasCond
            ? {op: condOp, value: condValue.trim(), ...(condOp === "between" ? {value2: condValue2.trim()} : {})}
            : null;
        if (!cond && checked === null) {
            onApplyFilter(null);
        } else {
            onApplyFilter({cond, checked});
        }
        onClose();
    };

    const handleClear = () => {
        onApplyFilter(null);
        onClose();
    };

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
        <div className={styles.popup}>
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
                               if (el) el.indeterminate = someChecked;
                           }}
                           checked={checked === null || allChecked}
                           onChange={() => setChecked(allChecked || checked === null ? [] : null)}/>
                    (Выбрать все)
                </label>
                <div className={styles.checkList}>
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
                </div>
                {distinct && distinct.truncated && (
                    <div className={styles.truncatedNote}>
                        Первые {distinct.data.length} из {distinct.total_distinct} — сузьте условием
                    </div>
                )}
            </div>

            <div className={styles.footer}>
                <button type="button" onClick={handleClear} className={styles.clearBtn}>
                    Сбросить
                </button>
                <button type="button" onClick={handleApply} className={styles.applyBtn}>
                    Применить
                </button>
            </div>
        </div>
    );
};
