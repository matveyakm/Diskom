import {FormField, RawColumnItem} from "@/entities/document/types";
import {TableColumnConfig} from "@/shared/DataTable/types";

export const parseTableColumns = (rawColumnsStr: string | undefined) => {
    if (!rawColumnsStr) return [];
    try {
        const rawJson = JSON.parse(rawColumnsStr) as { columns: RawColumnItem[] };

        if (rawJson && Array.isArray(rawJson.columns)) {
            return rawJson.columns.filter((c) => c.field !== "status").map((c) => ({
                key: c.field,
                label: c.label,
                type: "text",
            })) as TableColumnConfig[];
        }
        return [];
    } catch (e) {
        console.error("Критическая ошибка при JSON.parse(table_columns):", e);
        return [];
    }
};

export const parseFormFields = (rawFieldsStr: string | undefined): FormField[] => {
    if (!rawFieldsStr) return [];
    try {
        const rawJson = JSON.parse(rawFieldsStr) as { fields: FormField[] };

        if (rawJson && Array.isArray(rawJson.fields)) {
            return rawJson.fields;
        }
        return [];
    } catch (e) {
        console.error("Критическая ошибка при JSON.parse(item_fields):", e);
        return [];
    }
};
