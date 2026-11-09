  -- id_document БЕЗ end_point осознанно: "document_directory" не существует,
  -- а вид-документ меняется от строки к строке (нужен endpoint-aware expand,
  -- отдельная задача; см. MEMORY). Не копировать сюда document_directory.
  insert into ${DB_TABLENAME}
  (
    name,
    description,
    status,
    prefix,
    prefix_number,
    level,
    is_group,
    id_group,
    ref,
    output_order,
    icon,
    table_columns,
    table_title,
    item_title,
    item_fields,
    on_save_new,
    on_save_edit,
    on_acceptance,
    on_delete,
    privileges,
    related_documents,
    printing_template,
    id_transaction
  ) values (
    'Транзакции по реквизитам комитетов',
    'Транзакции по реквизитам комитетов',
    1,
    0,
    'ОРГ-',
    1,
    FALSE,
    4,
    'organization_data_ledger',
    4,
    'document.png',
    '{
      "columns": [
        { "field": "id_document", "label": "Документ"},
        { "field": "ddate", "label": "Дата"},
        { "field": "name", "label": "Наименование"},
        { "field": "id_organization", "label": "Комитет", "end_point": "organization_directory"}
      ]
    }',
    'Транзакции по реквизитам комитета',
    '',
    '',
    '',
    '',
    '',
    '',
    '{
      "privileges": [
        { "open_list": "true" },
        { "open_item": "true" },
        { "create": "false" },
        { "edit": "false" },
        { "delete": "false" },
        { "accept": "false" },
        { "create_related_documents": "false" }
      ]
    }',
    '',
    '',
    ''
  );
