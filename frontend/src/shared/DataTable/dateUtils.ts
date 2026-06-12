import {FormField} from "@/entities/document/types";

export const isDateField = (field: FormField): boolean => {
    if (field.type === "date") return true;
    if (field.field === "ddate") return true;
    if (field.field.toLowerCase().includes("date")) return true;
    return field.label?.toLowerCase().includes("дата");

};

export const formatDateForDisplay = (value: unknown): string => {
    if (!value || typeof value !== "string") return "—";
    try {
        const date = new Date(value);
        if (isNaN(date.getTime())) return String(value);
        return date.toLocaleDateString("ru-RU", {
            day: "2-digit",
            month: "2-digit",
            year: "numeric",
        });
    } catch {
        return String(value);
    }
};
