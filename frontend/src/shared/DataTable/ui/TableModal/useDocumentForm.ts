import React, {useState} from "react";
import {FormField} from "@/entities/document/types";
import {TableRow} from "@/shared/DataTable/types";
import {validateAndCollect} from "./formValidator";

type UseDocumentFormProps<T extends TableRow = TableRow> = {
    visibleFields: FormField[];
    mode: "edit" | "create";
    selectedRow: T | null;
    onSave: (payload: Record<string, unknown>) => void;
};

export const useDocumentForm = <T extends TableRow = TableRow>({
    visibleFields,
    mode,
    selectedRow,
    onSave,
}: UseDocumentFormProps<T>) => {
    const [errors, setErrors] = useState<Record<string, boolean>>({});
    const [isEditing, setIsEditing] = useState(mode === "create");

    const handleSubmit: React.SubmitEventHandler<HTMLFormElement> = (e) => {
        e.preventDefault();
        const formData = new FormData(e.currentTarget);
        const result = validateAndCollect(formData, visibleFields, mode, selectedRow);
        setErrors(result.errors);
        if (!result.payload) return;
        onSave(result.payload);
    };

    return {errors, setErrors, isEditing, setIsEditing, handleSubmit};
};
