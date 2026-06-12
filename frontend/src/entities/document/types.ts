export interface FormField {
    field: string;
    label: string;
    type: string;
    hidden?: boolean;
    required?: boolean;
    default?: unknown;
    end_point?: string;
}

export interface RawColumnItem {
    field: string;
    label: string;
}
