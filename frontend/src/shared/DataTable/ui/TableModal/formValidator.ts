import { FormField, FILE_DETACHED_SUFFIX } from "@/entities/document/types";
import { TableRow } from "@/shared/DataTable/types";
import { isDateField } from "./utils";

export const validateRequiredFields = (
    formData: FormData,
    fields: FormField[],
): Record<string, boolean> => {
    const errors: Record<string, boolean> = {};
    for (const field of fields) {
        if (field.required === false) continue;
        if (field.readonly) continue;
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
        if (field.type === "file") {
            // Файловое поле — виртуальное (колонки в строке нет): выборка id
            // файлов для шага привязки пайплайна. Пусто — [] (шаг — no-op).
            // Рядом — снятые крестиком (шаг detach; строка не удаляется).
            const all = formData.getAll(field.field)
                .map((v) => String(v))
                .filter((s) => s !== "")
                .map((s) => coerceSelectValue(s));
            typedFields[field.field] = all;
            const off = formData.getAll(`${field.field}${FILE_DETACHED_SUFFIX}`)
                .map((v) => String(v))
                .filter((s) => s !== "")
                .map((s) => coerceSelectValue(s));
            typedFields[`${field.field}${FILE_DETACHED_SUFFIX}`] = off;
            continue;
        }
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
