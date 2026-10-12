import {useEffect, useMemo, useState} from "react";
import {useParams} from "react-router-dom";
import styles from "./TablePage.module.scss";

import DataTable from "@/shared/DataTable";
import useMenuData from "@/shared/api/fetchMenuData";
import {useTableData} from "@/shared/api/fetchTableData";
import {parseFormBlocks, parseFormFields, parseTableColumns} from "@/shared/api/fetchMenuData/utils";
import {buildListParams} from "@/shared/DataTable/filters/queryParams";
import type {ColumnFilterState, ColumnSort, SortDir, TableColumnFilters} from "@/shared/DataTable/filters/types";

const TablePage = () => {
    const {docName} = useParams();
    // Другая таблица — новый экземпляр со чистым состоянием
    // (страница, сортировка, фильтры, поиск сбрасываются сами).
    return <TableView key={docName ?? "none"} docName={docName}/>;
};

const TableView = ({docName}: { docName: string | undefined }) => {
    const {data: menuData} = useMenuData();

    const tableMeta = useMemo(
        () => menuData?.find(item => item.ref === docName),
        [menuData, docName],
    );

    const tableType = useMemo((): 1 | 2 | null => {
        if (!tableMeta) return null;

        const groupId = tableMeta.id_group || tableMeta.id;

        if (groupId === 1) return 1;
        if (groupId === 2) return 2;

        return null;
    }, [tableMeta]);

    const [page, setPage] = useState(1);
    const PAGE_SIZE = 50;

    // Excel-фильтры: сортировка одна на таблицу, условия/галочки — по колонкам,
    // глобальный поиск — с debounce. Любое изменение сбрасывает на страницу 1.
    const [sort, setSort] = useState<ColumnSort | null>(null);
    const [columnFilters, setColumnFilters] = useState<TableColumnFilters>({});
    const [searchInput, setSearchInput] = useState("");
    const [search, setSearch] = useState("");

    // Глобальный поиск уходит в запрос с задержкой, чтобы не дёргать
    // сервер на каждую клавишу.
    useEffect(() => {
        const timer = setTimeout(() => setSearch(searchInput), 400);
        return () => clearTimeout(timer);
    }, [searchInput]);

    const columns = useMemo(
        () => parseTableColumns(tableMeta?.table_columns),
        [tableMeta?.table_columns]
    );

    // Reference expansion: columns carrying end_point ask the backend
    // to resolve ids into human-readable names (see shared/controller/expand.go).
    const expand = useMemo(
        () => columns
            .filter((c) => c.end_point)
            .map((c) => `${c.key}:${c.end_point}`)
            .join(",") || undefined,
        [columns]
    );

    // Глобальный поиск — только по локальным колонкам: имена *_name
    // резолвятся другими сервисами и на сервере недоступны.
    const searchFields = useMemo(
        () => columns.filter((c) => !c.end_point).map((c) => c.key),
        [columns]
    );

    const extraParams = useMemo(
        () => buildListParams({sort, filters: columnFilters, q: search, qFields: searchFields}),
        [sort, columnFilters, search, searchFields]
    );

    const hasActiveFilters =
        sort !== null ||
        search.trim() !== "" ||
        Object.values(columnFilters).some(s => s && (s.cond != null || s.checked != null));

    const handleSort = (key: string, dir: SortDir | null) => {
        setSort(dir ? {key, dir} : null);
        setPage(1);
    };

    const handleColumnFilter = (key: string, state: ColumnFilterState | null) => {
        setColumnFilters(prev => {
            if (state === null) {
                if (!(key in prev)) return prev;
                const next = {...prev};
                delete next[key];
                return next;
            }
            return {...prev, [key]: state};
        });
        setPage(1);
    };

    const handleClearFilters = () => {
        setSort(null);
        setColumnFilters({});
        setSearchInput("");
        setSearch("");
        setPage(1);
    };

    const {data: tableResponse, isLoading, isError, refetch} = useTableData(docName ?? "", page, PAGE_SIZE, expand, extraParams);

    const tableRowsResponse = tableResponse?.data ?? [];
    const pagination = tableResponse?.pagination;

    const formFields = useMemo(
        () => parseFormFields(tableMeta?.item_fields),
        [tableMeta?.item_fields]
    );

    const formBlocks = useMemo(
        () => parseFormBlocks(tableMeta?.item_fields),
        [tableMeta?.item_fields]
    );

    useEffect(() => {
        document.title = tableMeta?.table_title || "Просмотр таблицы";
    }, [tableMeta?.table_title]);

    if (isLoading || !tableType) {
        return (
            <div className={styles.wrapper}>
                <div className={styles.loader}>
                    <span className={styles.spinner}/>
                    <p className={styles.loaderText}>Загрузка данных...</p>
                </div>
            </div>
        );
    }

    if (isError) {
        return (
            <div className={styles.wrapper}>
                <div className={styles.error}>
                    <span className={styles.errorIcon}>⚠</span>
                    <p className={styles.errorText}>Не удалось загрузить данные</p>
                    <button className={styles.retryBtn} onClick={() => refetch()}>
                        Повторить
                    </button>
                </div>
            </div>
        );
    }

    return (
        <div className={styles.wrapper}>
            <DataTable
                columns={columns}
                rows={tableRowsResponse}
                docName={docName}
                fields={formFields}
                blocks={formBlocks}
                pageIndex={(pagination?.current_page ?? 1) - 1}
                pageCount={pagination?.total_pages ?? 1}
                totalCount={pagination?.total_count ?? 0}
                onPageChange={(p) => setPage(p + 1)}
                tableMeta={tableMeta}
                sort={sort}
                columnFilters={columnFilters}
                searchValue={searchInput}
                hasActiveFilters={hasActiveFilters}
                onSort={handleSort}
                onColumnFilter={handleColumnFilter}
                onSearchChange={(v) => {
                    setSearchInput(v);
                    setPage(1);
                }}
                onClearFilters={handleClearFilters}
            />
        </div>
    );
};

export default TablePage;
