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
    'Транзакции по реквизитам должностей',
    'Транзакции по реквизитам должностей',
    1,
    0,
    'ТД-',
    1,
    FALSE,
    4,
    'job_title_data_ledger',
    1,
    'document.png',
    '{
      "columns": [
        { "field": "id_document", "label": "Документ", "end_point": "document_directory"},
        { "field": "ddate", "label": "Дата"},
        { "field": "name", "label": "Наименование"},
        { "field": "id_job_title", "label": "Должность"}
      ]
    }',
    'Транзакции по реквизитам должности',
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
