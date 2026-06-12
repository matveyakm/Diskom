import { FormField } from "@/entities/document/types";
import { TableRow } from "@/shared/DataTable/types";
import { isDateField } from "./utils";

export const validateRequiredFields = (
    formData: FormData,
    fields: FormField[],
): Record<string, boolean> => {
    const errors: Record<string, boolean> = {};
    for (const field of fields) {
        if (field.required === false) continue;
        if (field.default !== undefined) continue;
        const value = formData.get(field.field);
        if (value === null || String(value).trim() === "") {
            errors[field.field] = true;
        }
    }
    return errors;
};

const coerceSelectValue = (value: string): string | number => {
    const n = Number(value);
    return isNaN(n) ? value : n;
};

export const collectFormData = (
    formData: FormData,
    fields: FormField[],
    mode: "edit" | "create",
    selectedRow: TableRow | null,
): Record<string, unknown> => {
    const typedFields: Record<string, unknown> = {};

    for (const field of fields) {
        const value = formData.get(field.field);
        if (value === null || value === "") {
            typedFields[field.field] = null;
            continue;
        }
        typedFields[field.field] =
            field.type === "select" && field.end_point
                ? coerceSelectValue(String(value))
                : String(value);
    }

    for (const field of fields) {
        const val = typedFields[field.field];
        if (isDateField(field) && typeof val === "string" && /^\d{4}-\d{2}-\d{2}$/.test(val)) {
            typedFields[field.field] = val + "T00:00:00.000Z";
        }
    }

    const selectedRowData = (selectedRow ?? {}) as Record<string, unknown>;
    const payload: Record<string, unknown> = mode === "create"
        ? { status: 1, ...typedFields }
        : {
            ...selectedRowData, ...typedFields,
            id: selectedRow?.id
        };

    return payload;
};

export const validateAndCollect = (
    formData: FormData,
    fields: FormField[],
    mode: "edit" | "create",
    selectedRow: TableRow | null,
): { errors: Record<string, boolean>; payload: Record<string, unknown> } | { errors: Record<string, boolean>; payload: null } => {
    const errors = validateRequiredFields(formData, fields);
    if (Object.keys(errors).length > 0) {
        return { errors, payload: null };
    }

    const payload = collectFormData(formData, fields, mode, selectedRow);
    return { errors: {}, payload };
};
