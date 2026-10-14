import {FormBlock, FormField} from "@/entities/document/types";
import {MenuItem} from "@/shared/api/fetchMenuData/types";
import type {ColumnSort, TableColumnFilters} from "@/shared/DataTable/filters/types";

export interface TableColumnConfig {
    key: string;
    label: string;
    type: "text" | "status";
    end_point?: string;
}

export interface TableRow {
    id: number | string;
}

export type DocumentsTableProps<T extends TableRow = TableRow> = {
    columns: TableColumnConfig[];
    rows: T[];
    docName: string | undefined;
    fields: FormField[];
    blocks?: FormBlock[];
    pageIndex: number;
    pageCount: number;
    totalCount: number;
    onPageChange: (page: number) => void;
    tableMeta?: MenuItem;
    /** Excel-фильтры; окно открывается кнопкой тулбара, коммит — одной кнопкой. */
    sort?: ColumnSort | null;
    columnFilters?: TableColumnFilters;
    hasActiveFilters?: boolean;
    onApplyFilters?: (sort: ColumnSort | null, filters: TableColumnFilters) => void;
    onClearFilters?: () => void;
};