import {flexRender} from "@tanstack/react-table";
import styles from "./DataTable.module.scss";
import {useTable} from "@/shared/DataTable/hooks/useTable";
import CustomToolbar from "@/shared/DataTable/ui/CustomToolbar";
import {useModal} from "@/shared/DataTable/hooks/useModal";
import {useContextMenu} from "@/shared/DataTable/hooks/useContextMenu";
import {useRowSelection} from "@/shared/DataTable/hooks/useRowSelection";
import React, {useRef} from "react";
import {useOnClickOutside} from "usehooks-ts";
import TableModal from "@/shared/DataTable/ui/TableModal";
import ContextMenu from "@/shared/DataTable/ui/ContextMenu";
import Pagination from "./ui/Pagination";
import { asElementRef } from "@/shared/utils/refUtils";
import {DocumentsTableProps, TableRow} from "@/shared/DataTable/types";

const DataTable = <T extends TableRow = TableRow>({
                       columns,
                       rows,
                       docName,
                       fields,
                       pageIndex,
                       pageCount,
                       onPageChange,
                       tableMeta
                   }: DocumentsTableProps<T>) => {
    const {table} = useTable<T>({columns, rows, fields});
    const {isOpen, data: selectedRow, mode, openForEdit, openForCreate, close} = useModal<T>();
    const {menu, openMenu, closeMenu} = useContextMenu<T>();
    const {activeRow, selectRow, clearSelection} = useRowSelection<T>();
    const tableName = tableMeta?.table_title;

    const wrapperRef = useRef<HTMLDivElement>(null);

    useOnClickOutside(asElementRef(wrapperRef), () => {
        clearSelection();
        closeMenu();
    });

    const onEdit = () => {
        if (activeRow) {
            openForEdit(activeRow);
            window.getSelection()?.removeAllRanges();
        }
    };

    return (
        <div className={styles.wrapper} ref={wrapperRef}>
            <div className={styles.tableContainer}>
                <CustomToolbar
                    selectedRow={activeRow}
                    onAdd={openForCreate}
                    onEdit={onEdit}
                    tableName={tableName}
                />
                <div className={styles.tableScroll}>
                    <table className={styles.table}>
                        <thead>
                        {table.getHeaderGroups().map(headerGroup => (
                            <tr key={headerGroup.id}>
                                {headerGroup.headers.map(header => (
                                    <th key={header.id} style={{width: header.getSize(), position: 'relative'}}>
                                        <div className={styles.headerContent}>
                                            {flexRender(
                                                header.column.columnDef.header,
                                                header.getContext()
                                            )}
                                        </div>
                                        <div
                                            onMouseDown={header.getResizeHandler()}
                                            onTouchStart={header.getResizeHandler()}
                                            className={styles.resizer}
                                        />
                                    </th>
                                ))}
                            </tr>
                        ))}
                        </thead>

                        <tbody>
                        {table.getRowModel().rows.length === 0 ? (
                            <tr>
                                <td colSpan={columns.length} className={styles.emptyCell}>
                                    Нет данных для отображения
                                </td>
                            </tr>
                        ) : (
                            table.getRowModel().rows.map(row => (
                                <tr
                                    key={row.id}
                                    className={`${styles.row} ${activeRow?.id === row.original.id ? styles.rowActive : ""}`}
                                    onClick={(e) => {
                                        e.stopPropagation();
                                        selectRow(row.original as T);
                                    }}
                                    onDoubleClick={() => {
                                        openForEdit(row.original as T);
                                        window.getSelection()?.removeAllRanges();
                                    }}
                                    onContextMenu={(e) => {
                                        selectRow(row.original as T);
                                        openMenu(e, row.original as T);
                                    }}
                                >
                                    {row.getVisibleCells().map(cell => (
                                        <td key={cell.id} style={{width: cell.column.getSize()}}>
                                            <div className={styles.cellEllipsis}>
                                                {flexRender(
                                                    cell.column.columnDef.cell,
                                                    cell.getContext()
                                                )}
                                            </div>
                                        </td>
                                    ))}
                                </tr>
                            ))
                        )}
                        </tbody>
                    </table>
                </div>
            </div>

            <div className={styles.footer}>
                <Pagination
                    pageIndex={pageIndex}
                    pageCount={pageCount}
                    onPageChange={onPageChange}
                />
            </div>

            {isOpen && selectedRow && (
                <TableModal
                    selectedRow={selectedRow}
                    onClose={close}
                    docName={docName ?? "Документ"}
                    fields={fields ?? []}
                    mode={mode}
                    tableMeta={tableMeta}
                />
            )}

            {menu.visible && (<ContextMenu menu={menu} onClose={closeMenu} onEdit={onEdit}/>)}

        </div>
    );
};

export default DataTable;