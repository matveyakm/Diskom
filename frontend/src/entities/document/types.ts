export interface FormField {
    field: string;
    label: string;
    type: string;
    hidden?: boolean;
    required?: boolean;
    readonly?: boolean;
    default?: unknown;
    end_point?: string;
}

export interface RawColumnItem {
    field: string;
    label: string;
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
