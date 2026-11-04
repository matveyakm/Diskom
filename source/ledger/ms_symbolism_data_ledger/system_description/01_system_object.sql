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
    'Транзакции по символике',
    'Транзакции по символике',
    1,
    0,
    'СИ-',
    1,
    FALSE,
    4,
    'symbolism_data_ledger',
    3,
    'document.png',
    '{
      "columns": [
        { "field": "id_document", "label": "Документ", "end_point": "symbolism_document"},
        { "field": "ddate", "label": "Дата"},
        { "field": "id_committee", "label": "Отдел", "end_point": "department_directory"},
        { "field": "comment", "label": "Комментарий"}
      ]
    }',
    'Журнал транзакций по символике',
    'Транзакция по символике',
    '{
  "fields": [
    {
      "field": "id",
      "label": "",
      "hidden": true,
      "required": false
    },
    {
      "field": "id_document",
      "label": "Документ",
      "hidden": false,
      "required": true
    },
    {
      "field": "ddate",
      "label": "Дата",
      "hidden": false,
      "required": false
    }
  ]
}',
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
