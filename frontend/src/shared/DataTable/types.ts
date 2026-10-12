import {FormBlock, FormField} from "@/entities/document/types";
import {MenuItem} from "@/shared/api/fetchMenuData/types";
import type {ColumnFilterState, ColumnSort, SortDir, TableColumnFilters} from "@/shared/DataTable/filters/types";

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
    /** Excel-фильтры; отсутствуют — грид как раньше, без попапов. */
    sort?: ColumnSort | null;
    columnFilters?: TableColumnFilters;
    searchValue?: string;
    hasActiveFilters?: boolean;
    onSort?: (key: string, dir: SortDir | null) => void;
    onColumnFilter?: (key: string, state: ColumnFilterState | null) => void;
    onSearchChange?: (value: string) => void;
    onClearFilters?: () => void;
};