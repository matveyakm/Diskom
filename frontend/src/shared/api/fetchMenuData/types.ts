export interface MenuItem {
    id: number;
    name: string;
    description: string;
    status: number;
    basis: number;
    prefix: number;
    level: number;
    is_group: boolean;
    id_group: number;
    ref: string;
    output_order: number;
    icon: string;
    table_columns?: string;
    table_title?: string;
    item_title?: string;
    item_fields?: string;
    inheritance?: string;
    printing_template?: string;
    on_save_new?: string;
    on_save_edit?: string;
    on_acceptance?: string;
    on_delete?: string;
}
