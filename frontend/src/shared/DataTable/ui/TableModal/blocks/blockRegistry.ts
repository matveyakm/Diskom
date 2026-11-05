import {ComponentType} from "react";
import {FormBlock, FormField} from "@/entities/document/types";
import {DeclensionTable} from "./DeclensionTable";

// Контракт кастомного рендерера блока:
// - рисует ТОЛЬКО представление, данные не меняет;
// - каждое поле блока обязано попасть в форму под своим name=
//   (проще всего — через bare <FieldRow>: там уже hidden-инпуты и required),
//   иначе значение молча сохранится как null (ловит dev-проверка в useDocumentForm);
// - неизвестные теги сюда не попадают (см. resolveBlockRenderer).
export type BlockRenderProps = {
    block: FormBlock;
    fields: FormField[];
    getValue: (name: string) => unknown;
    isEditing: boolean;
    errors: Record<string, boolean>;
    /** ref текущей сущности (docName === ref) для файловых URL /api/<ref>/files */
    serviceRef?: string;
    /** id строки — для списка файлов документа */
    docId?: number | string;
};

const registry: Record<string, ComponentType<BlockRenderProps>> = {
    "declension-table": DeclensionTable,
};

export const resolveBlockRenderer = (tag?: string): ComponentType<BlockRenderProps> | null => {
    if (!tag) return null;
    return registry[tag] ?? null;
};
