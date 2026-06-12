import {TableRow} from "@/shared/DataTable/types";
import {isDateField, formatDateForDisplay} from "@/shared/DataTable/dateUtils";

export {isDateField, formatDateForDisplay};

export const getRowValue = (
    selectedRow: TableRow | null,
    fieldKey: string,
): unknown => {
    if (!selectedRow || typeof selectedRow !== "object") return undefined;
    return (selectedRow as unknown as Record<string, unknown>)[fieldKey];
};

const pad = (n: number): string => String(n).padStart(2, "0");

const toLocalYmd = (date: Date): string =>
    `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;

export const toDateInputValue = (value: unknown): string => {
    if (!value || typeof value !== "string") return "";
    const match = value.match(/^(\d{4}-\d{2}-\d{2})/);
    if (match) return match[1];
    try {
        const date = new Date(value);
        if (isNaN(date.getTime())) return "";
        return toLocalYmd(date);
    } catch {
        return "";
    }
};

export const getTodayDate = (): string => toLocalYmd(new Date());

