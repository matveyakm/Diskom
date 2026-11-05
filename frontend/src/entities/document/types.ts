export interface FormField {
    field: string;
    label: string;
    type: string;
    hidden?: boolean;
    required?: boolean;
    readonly?: boolean;
    default?: unknown;
    end_point?: string;
    /** Ограничение числа файлов для type: "file" (жёсткий гейт выборки: сверх лимита загрузить нельзя). */
    max_count?: number;
}

/** Суффикс виртуального поля «снятые с выборки»: FileField пишет туда id
    файлов, убранных крестиком (для шага detach пайплайна); строка файлов
    при этом НЕ удаляется — станет сиротой, подберёт GC. */
export const FILE_DETACHED_SUFFIX = "_detached";

export interface RawColumnItem {
    field: string;
    label: string;
    type?: string;
    end_point?: string;
}

export interface FormBlockRow {
    label: string;
    fields: string[];
}

export interface FormBlock {
    id?: string;
    title: string;
    tag?: string;
    fields: string[];
    columns?: string[];
    rows?: FormBlockRow[];
}

export interface RawFormItem {
    fields: FormField[];
    blocks?: FormBlock[];
}
