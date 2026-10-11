import {FormBlock, FormField} from "@/entities/document/types";
import {MenuItem} from "@/shared/api/fetchMenuData/types";

export interface TableColumnConfig {
    key: string;
    label: string;
    type: "text";
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
};