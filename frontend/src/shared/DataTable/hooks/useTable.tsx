import {useMemo} from "react";
import {ColumnDef, getCoreRowModel, useReactTable} from "@tanstack/react-table";
import {TableRow, TableColumnConfig} from "@/shared/DataTable/types";
import {FormField} from "@/entities/document/types";
import {isDateField, formatDateForDisplay} from "@/shared/DataTable/dateUtils";
import {StatusIcon} from "@/shared/DataTable/ui/StatusIcon";
import {HeaderCell} from "@/shared/DataTable/ui/FilterPopup";
import type {HeaderFiltering} from "@/shared/DataTable/filters/types";

interface UseTableProps<T extends TableRow = TableRow> {
    columns: TableColumnConfig[];
    rows: T[];
    fields?: FormField[];
    filtering?: HeaderFiltering;
}

export const useTable = <T extends TableRow = TableRow>({columns, rows, fields, filtering}: UseTableProps<T>) => {
    const dateFieldKeys = useMemo(() => {
        if (!fields) return new Set<string>();
        return new Set(fields.filter(isDateField).map(f => f.field));
    }, [fields]);

    const fieldByName = useMemo(() => {
        const map = new Map<string, FormField>();
        for (const f of fields ?? []) map.set(f.field, f);
        return map;
    }, [fields]);

    const columnDefs = useMemo<ColumnDef<Record<string, unknown>>[]>(() => {
        return columns.map(col => {
            const columnKey = col.key;
            const isDate = dateFieldKeys.has(columnKey);

            return {
                id: String(columnKey),
                accessorKey: columnKey,
                header: () => {
                    if (!filtering) return col.label || columnKey;
                    const key = String(columnKey);
                    return (
                        <HeaderCell
                            column={col}
                            formField={fieldByName.get(key)}
                            docName={filtering.docName}
                            sortDir={filtering.sort?.key === key ? filtering.sort.dir : null}
                            filter={filtering.filters[key] ?? null}
                            isOpen={filtering.openKey === key}
                            onToggle={() => filtering.onTogglePopup(filtering.openKey === key ? null : key)}
                            onClose={() => filtering.onTogglePopup(null)}
                            onSort={(dir) => filtering.onSort(key, dir)}
                            onApplyFilter={(state) => filtering.onFilter(key, state)}
                        />
                    );
                },
                cell: ({getValue, row}) => {
                    if (col.type === "status") return <StatusIcon value={getValue()}/>;
                    // Prefer backend-resolved display name (see ?expand=),
                    // fall back to the raw value when absent.
                    const resolved = row.original[`${String(columnKey)}_name`];
                    if (typeof resolved === "string" && resolved !== "") return resolved;
                    const value = getValue();
                    if (value === undefined || value === null) return "";
                    if (isDate) return formatDateForDisplay(value);
                    return String(value);
                }
            };
        });
    }, [columns, dateFieldKeys, fieldByName, filtering]);

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