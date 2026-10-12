import {createPortal} from "react-dom";
import {useMemo, useState} from "react";
import styles from "./FiltersModal.module.scss";
import {TableColumnConfig} from "@/shared/DataTable/types";
import {FormField} from "@/entities/document/types";
import {FilterEditor} from "@/shared/DataTable/ui/FilterEditor";
import {buildListParams} from "@/shared/DataTable/filters/queryParams";
import type {ColumnFilterState, ColumnSort, ConditionFilter, SortDir, TableColumnFilters} from "@/shared/DataTable/filters/types";

interface FiltersModalProps {
    docName: string | undefined;
    columns: TableColumnConfig[];
    /** Все поля сущности (включая скрытые), не только колонки грида. */
    formFields: FormField[];
    sort: ColumnSort | null;
    filters: TableColumnFilters;
    onApply: (sort: ColumnSort | null, filters: TableColumnFilters) => void;
    onClose: () => void;
}

const isActive = (s: ColumnFilterState | undefined): boolean =>
    !!s && (s.cond != null || s.checked != null);

// Закладки: пользовательские поля (hidden — технические, не показываем)
// плюс колонки грида, которых нет в полях (например status).
const tabFields = (formFields: FormField[], columns: TableColumnConfig[]): FormField[] => {
    const visible = formFields.filter(f => !f.hidden);
    const known = new Set(visible.map(f => f.field));
    const extra: FormField[] = columns
        .filter(c => !known.has(String(c.key)))
        .map(c => ({field: String(c.key), label: c.label || String(c.key), type: ""}));
    return [...visible, ...extra];
};

// Окно фильтров: слева закладки всех полей сущности, справа редактор
// выбранного поля. Черновик живёт здесь и коммитится одной кнопкой
// «Применить»; закрытие без неё ничего не меняет.
export const FiltersModal = ({
    docName, columns, formFields, sort, filters, onApply, onClose,
}: FiltersModalProps) => {
    const [draftSort, setDraftSort] = useState<ColumnSort | null>(sort);
    const [draftFilters, setDraftFilters] = useState<TableColumnFilters>(filters);

    // Закладки: пользовательские поля (hidden — технические, не показываем)
    // плюс колонки грида, которых нет в полях (например status).
    const tabs = useMemo<FormField[]>(() => tabFields(formFields, columns), [formFields, columns]);

    const [tabKey, setTabKey] = useState<string>(() => {
        const all = tabFields(formFields, columns);
        return all.find(f => isActive(filters[f.field]))?.field ?? all[0]?.field ?? "";
    });

    const columnByKey = new Map(columns.map(c => [String(c.key), c]));
    const tab = tabs.find(f => f.field === tabKey);

    // Параметры соседних закладок для сужения списка галочек: всё из
    // черновика, кроме собственного столбца (его бэкенд всё равно вычитает).
    const siblingParams = useMemo(() => {
        if (!tab) return {};
        const siblings: TableColumnFilters = {...draftFilters};
        delete siblings[tab.field];
        return buildListParams({filters: siblings});
    }, [draftFilters, tab]);

    const writeFilter = (key: string, state: ColumnFilterState | null) => {
        setDraftFilters(prev => {
            if (state === null || (state.cond == null && state.checked == null)) {
                if (!(key in prev)) return prev;
                const next = {...prev};
                delete next[key];
                return next;
            }
            return {...prev, [key]: state};
        });
    };

    const draftFor = (key: string): ColumnFilterState | null => draftFilters[key] ?? null;

    const handleSort = (key: string, dir: SortDir | null) => {
        setDraftSort(dir ? {key, dir} : null);
    };

    const handleResetAll = () => {
        setDraftSort(null);
        setDraftFilters({});
    };

    return createPortal(
        <div className={styles.overlay} onClick={onClose}>
            <div className={styles.window} onClick={(e) => e.stopPropagation()}>
                <header className={styles.header}>
                    <h3 className={styles.title}>Фильтры</h3>
                    <button className={styles.closeX} onClick={onClose} aria-label="Закрыть">
                        ✕
                    </button>
                </header>
                <div className={styles.body}>
                    <nav className={styles.tabs}>
                        {tabs.map(f => {
                            const st = draftFilters[f.field];
                            return (
                                <button
                                    key={f.field}
                                    type="button"
                                    className={`${styles.tab} ${f.field === tabKey ? styles.tabActive : ""}`}
                                    onClick={() => setTabKey(f.field)}
                                >
                                    <span className={styles.tabLabel}>{f.label || f.field}</span>
                                    {draftSort?.key === f.field && (
                                        <span>{draftSort.dir === "asc" ? " ▲" : " ▼"}</span>
                                    )}
                                    {isActive(st) && <span className={styles.tabDot}> ●</span>}
                                </button>
                            );
                        })}
                    </nav>
                    <div className={styles.pane}>
                        {tab ? (
                            <FilterEditor
                                key={tab.field}
                                docName={docName}
                                fieldKey={tab.field}
                                column={columnByKey.get(tab.field)}
                                formField={tab}
                                sortDir={draftSort?.key === tab.field ? draftSort.dir : null}
                                cond={draftFor(tab.field)?.cond ?? null}
                                checked={draftFor(tab.field)?.checked ?? null}
                                siblingParams={siblingParams}
                                onSort={(dir) => handleSort(tab.field, dir)}
                                onCond={(cond: ConditionFilter | null) => {
                                    const cur = draftFor(tab.field);
                                    writeFilter(tab.field, {cond, checked: cur?.checked ?? null});
                                }}
                                onChecked={(values) => {
                                    const cur = draftFor(tab.field);
                                    writeFilter(tab.field, {cond: cur?.cond ?? null, checked: values});
                                }}
                                onResetField={() => writeFilter(tab.field, null)}
                            />
                        ) : (
                            <div className={styles.noFields}>Нет полей</div>
                        )}
                    </div>
                </div>
                <footer className={styles.footer}>
                    <button type="button" onClick={handleResetAll} className={styles.clearBtn}>
                        Сбросить всё
                    </button>
                    <button type="button"
                            onClick={() => {
                                onApply(draftSort, draftFilters);
                                onClose();
                            }}
                            className={styles.applyBtn}>
                        Применить
                    </button>
                </footer>
            </div>
        </div>,
        document.body,
    );
};
