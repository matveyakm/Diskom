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
    'Остатки штата',
    'Остатки штата',
    1,
    0,
    'ТР-',
    1,
    FALSE,
    4,
    'employee_data_ledger',
    1,
    'document.png',
    '{
      "columns": [
        { "field": "id_document", "label": "Документ", "end_point": "document_directory"},
        { "field": "ddate", "label": "Дата"},
        { "field": "endpoint", "label": "Тип документа"},
        { "field": "id_employee", "label": "Сотрудник", "end_point": "employee_directory"},
        { "field": "delta", "label": "Изм."},
        { "field": "balance", "label": "Баланс"},
        { "field": "fio", "label": "ФИО на дату"}
      ]
    }',
    'Остатки штата',
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
