export interface FormField {
    field: string;
    label: string;
    type: string;
    hidden?: boolean;
    required?: boolean;
    readonly?: boolean;
    default?: unknown;
    /** @deprecated use editor/editor_ref (legacy JSON blobs) */
    end_point?: string;
    /** New shape: how the field is edited (default "manual"). */
    editor?: "manual" | "dropdown" | "form";
    /** New shape: reference target (system_object link path) for dropdown/form editors. */
    editor_ref?: string;
}

/** Reference target for select-like fields: new editor_ref wins, legacy end_point falls back. */
export const resolveRefEndpoint = (f: Pick<FormField, "end_point" | "editor" | "editor_ref"> | undefined): string | undefined =>
    f?.editor_ref ?? f?.end_point;

/** True for dropdown/form editors (new) or legacy type:"select"+end_point. */
export const isReferenceField = (f: Pick<FormField, "type" | "end_point" | "editor" | "editor_ref"> | undefined): boolean => {
    if (!f) return false;
    if (f.editor === "dropdown" || f.editor === "form") return true;
    return f.type === "select" && !!resolveRefEndpoint(f);
};

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
