import {useMemo} from "react";
import {ColumnDef, getCoreRowModel, useReactTable} from "@tanstack/react-table";
import {TableRow, TableColumnConfig} from "@/shared/DataTable/types";
import {FormField} from "@/entities/document/types";
import {isDateField, formatDateForDisplay} from "@/shared/DataTable/dateUtils";

interface UseTableProps<T extends TableRow = TableRow> {
    columns: TableColumnConfig[];
    rows: T[];
    fields?: FormField[];
}

export const useTable = <T extends TableRow = TableRow>({columns, rows, fields}: UseTableProps<T>) => {
    const dateFieldKeys = useMemo(() => {
        if (!fields) return new Set<string>();
        return new Set(fields.filter(isDateField).map(f => f.field));
    }, [fields]);

    const columnDefs = useMemo<ColumnDef<Record<string, unknown>>[]>(() => {
        return columns.map(col => {
            const columnKey = col.key;
            const isDate = dateFieldKeys.has(columnKey);

            return {
                id: String(columnKey),
                accessorKey: columnKey,
                header: col.label || columnKey,
                cell: ({getValue}) => {
                    const value = getValue();
                    if (value === undefined || value === null) return "";
                    if (isDate) return formatDateForDisplay(value);
                    return String(value);
                }
            };
        });
    }, [columns, dateFieldKeys]);

    const table = useReactTable({
        data: rows as Record<string, unknown>[],
        columns: columnDefs,
        getCoreRowModel: getCoreRowModel(),
        enableColumnResizing: true,
        columnResizeMode: 'onChange',
        defaultColumn: {
            size: 100,
            minSize: 20,
        },
    });

    return {table};
};