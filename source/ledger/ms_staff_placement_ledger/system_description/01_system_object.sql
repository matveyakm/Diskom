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
    'Транзакции изменеия штатного расписания',
    'Транзакции изменеия штатного расписания',
    1,
    0,
    'ЭТР-',
    1,
    FALSE,
    4,
    'staff_placement_ledger',
    2,
    'document.png',
    '{
      "columns": [
        { "field": "balance", "label": "", "hidden": false},
        { "field": "id_document", "label": "Документ", "end_point": "document_directory", "hidden": false},
        { "field": "ddate", "label": "Дата", "hidden": false},
        { "field": "id_employee", "label": "Сотрудник", "hidden": false, "end_point": "employee_directory"},
        { "field": "id_organization", "label": "Организация", "hidden": false, "end_point": "organization_directory"},
        { "field": "id_department", "label": "Отдел", "hidden": false, "end_point": "department_directory"},
        { "field": "id_job_title", "label": "Должность", "hidden": false, "end_point": "job_title_directory"},
        { "field": "justification", "label": "Обоснование", "hidden": false}
      ]
    }',
    'Транзакции изменеия штатного расписания',
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
