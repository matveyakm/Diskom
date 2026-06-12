import {FormField} from "@/entities/document/types";
import {TableRow} from "@/shared/DataTable/types";
import {MenuItem} from "@/shared/api/fetchMenuData/types";

export type TableModalProps<T extends TableRow = TableRow> = {
    selectedRow: T | null;
    onClose: () => void;
    docName: string;
    fields: FormField[];
    mode: "edit" | "create";
    tableMeta?: MenuItem;
};

export type FieldRowProps = {
    field: FormField;
    value: unknown;
    isEditing: boolean;
    error?: boolean;
};
