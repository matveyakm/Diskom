/** Single catalog object: one row of the ${DB_TABLENAME} view
 * (db.sh.base) = one joined system_object. JSON blobs stay strings;
 * use parse* helpers in ./utils to decode them. */
export interface MenuItem {
    id: number;
    name: string;
    description: string;
    status: number;
    prefix: number;
    prefix_number?: string;
    level: number;
    is_group: boolean;
    id_group: number | null;
    ref: string;
    output_order: number;
    icon: string;
    object_type_name?: string;
    object_status_name?: string;
    business_processes?: string;
    table_columns?: string;
    table_title?: string;
    item_title?: string;
    item_fields?: string;
    printing_template?: string;
    on_save_new?: string;
    on_save_edit?: string;
    on_acceptance?: string;
    on_delete?: string;
    privileges?: string;
    related_documents?: string;
}
