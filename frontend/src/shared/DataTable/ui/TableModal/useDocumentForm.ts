import React, {useState} from "react";
import {FormField} from "@/entities/document/types";
import {TableRow} from "@/shared/DataTable/types";
import {validateAndCollect} from "./formValidator";

type UseDocumentFormProps<T extends TableRow = TableRow> = {
    visibleFields: FormField[];
    mode: "edit" | "create";
    selectedRow: T | null;
    onSave: (payload: Record<string, unknown>, action: SubmitAction) => void;
};

export type SubmitAction = "save" | "saveAccept";

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
        // Какая кнопка отправила форму (у "Сохранить и провести" data-action).
        // Enter и обычная кнопка дают "save". Читаем здесь, а не через ref,
        // чтобы неудачная валидация не "залипала" на следующем сабмите.
        const submitter = (e.nativeEvent as SubmitEvent).submitter as HTMLElement | null;
        const action: SubmitAction = submitter?.dataset?.action === "saveAccept" ? "saveAccept" : "save";
        const formData = new FormData(e.currentTarget);
        // Ловит кастомные рендереры блоков, забывшие name= у инпутов:
        // такое поле молча сохранилось бы как null.
        for (const field of visibleFields) {
            if (!formData.has(field.field)) {
                console.error(`[form] поле "${field.field}" отсутствует в форме — значение не сохранится`);
            }
        }
        const result = validateAndCollect(formData, visibleFields, mode, selectedRow);
        setErrors(result.errors);
        if (!result.payload) return;
        onSave(result.payload, action);
    };

    return {errors, setErrors, isEditing, setIsEditing, handleSubmit};
};
