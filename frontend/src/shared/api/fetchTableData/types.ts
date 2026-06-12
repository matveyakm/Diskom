export interface DirectoryItem {
    id: number;
    name: string;
    description: string;
    status: number;
    basis: number;
    prefix: number;
    id_transaction: string;
    first_name?: string;
    last_name?: string;
    middle_name?: string;
    level?: number;
    is_group?: boolean;
    id_group?: number;
    ref?: string;
    output_order?: number;
    icon?: string;
    table_columns?: string;
    table_title?: string;
    item_title?: string;
    item_fields?: string;
    inheritance?: string;
    printing_template?: string;
}

export interface DocumentItem {
    id: number;
    ddate: string;
    dnumber: string;
    note: string;
    status: number;
    duser: number;
    prefix: number;
    id_document?: number;
    id_transaction?: string;
    id_system_object?: number;
    first_name?: string;
    last_name?: string;
    middle_name?: string;
    first_genitive?: string;
    last_genitive?: string;
    middle_genitive?: string;
    first_accusative?: string;
    last_accusative?: string;
    middle_accusative?: string;
    id_vk?: string;
    id_employee?: number;
    id_organization?: number;
    id_department?: number;
    id_job_title?: number;
    justification?: string;
}

export interface PaginationMeta {
    current_page: number;
    page_size: number;
    total_pages: number;
    total_count: number;
    has_next: boolean;
    has_previous: boolean;
    offset: number;
    limit: number;
}

export interface PaginatedResponse<T> {
    data: T[];
    pagination: PaginationMeta;
}

export interface DocumentItem extends Record<string, unknown> {
    id: number;
    ddate: string;
    dnumber: string;
    note: string;
    status: number;
    duser: number;
    prefix: number;
    id_document?: number;
    id_system_object?: number;
    first_name?: string;
    last_name?: string;
    middle_name?: string;
    first_genitive?: string;
    last_genitive?: string;
    middle_genitive?: string;
    first_accusative?: string;
    last_accusative?: string;
    middle_accusative?: string;
    id_vk?: string;
    id_employee?: number;
    id_organization?: number;
    id_department?: number;
    id_job_title?: number;
    justification?: string;
}

export interface PaginationMeta extends Record<string, unknown> {
    current_page: number;
    page_size: number;
    total_pages: number;
    total_count: number;
    has_next: boolean;
    has_previous: boolean;
    offset: number;
    limit: number;
}

export interface PaginatedResponse<T> extends Record<string, unknown> {
    data: T[];
    pagination: PaginationMeta;
}
