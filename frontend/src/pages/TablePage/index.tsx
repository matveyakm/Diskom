import {useEffect, useMemo, useState} from "react";
import {useParams} from "react-router-dom";
import styles from "./TablePage.module.scss";

import DataTable from "@/shared/DataTable";
import useMenuData from "@/shared/api/fetchMenuData";
import {useTableData} from "@/shared/api/fetchTableData";
import {parseFormFields, parseTableColumns} from "@/shared/api/fetchMenuData/utils";

const TablePage = () => {
    const {docName} = useParams();
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

    const {data: tableResponse, isLoading, isError, refetch} = useTableData(docName ?? "", page, PAGE_SIZE);

    const tableRowsResponse = tableResponse?.data ?? [];
    const pagination = tableResponse?.pagination;

    const columns = useMemo(
        () => parseTableColumns(tableMeta?.table_columns),
        [tableMeta?.table_columns]
    );

    const formFields = useMemo(
        () => parseFormFields(tableMeta?.item_fields),
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
                pageIndex={(pagination?.current_page ?? 1) - 1}
                pageCount={pagination?.total_pages ?? 1}
                totalCount={pagination?.total_count ?? 0}
                onPageChange={(p) => setPage(p + 1)}
                tableMeta={tableMeta}
            />
        </div>
    );
};

export default TablePage;