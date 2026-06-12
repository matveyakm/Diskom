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
    'Снятие с должности',
    'Снятие с должности',
    1,
    0,
    'СН-',
    1,
    FALSE,
    2,
    'dismissal_document',
    3,
    'document.png',
    '{
      "columns": [
        { "field": "status", "label": ""},
        { "field": "id", "label": "Код" },
        { "field": "ddate", "label": "Дата"},
        { "field": "dnumber", "label": "Номер"},
        { "field": "note", "label": "Примечание"},
        { "field": "id_department", "label": "Отдел", "end_point": "department_directory"},
        { "field": "id_job_title", "label": "Должность", "end_point": "job_title_directory"}
      ]
    }',
    'Журнал документов снятия с должности',
    'Документ снятия с должности',
    '{
      "fields": [
        { "field": "id", "label": "Код", "hidden": true, "required": false },
        { "field": "ddate", "label": "Дата", "hidden": false, "required": true, "default": "now()" },
        { "field": "dnumber", "label": "Номер", "hidden": false, "required": true, "default": "id" },
        { "field": "note", "label": "Примечание", "hidden": false, "required": false },
        { "field": "id_employee", "label": "Сотрудник", "hidden": false, "required": false, "type": "select", "end_point": "employee_directory" },
        { "field": "id_organization", "label": "Комитет", "hidden": false, "required": false, "type": "select", "end_point": "organization_directory" },
        { "field": "id_department", "label": "Отдел", "hidden": false, "required": false, "type": "select", "end_point": "department_directory" },
        { "field": "id_job_title", "label": "Должность", "hidden": false, "required": false, "type": "select", "end_point": "job_title_directory" },
        { "field": "justification", "label": "Обоснование", "hidden": false, "required": false },
        { "field": "id_user", "label": "IdUser", "hidden": true, "required": false, "default": 1 },
        { "field": "id_basis", "label": "IdBasis", "hidden": true, "required": false, "default": 0 }
      ]
    }',
    '{
        "action": "onSaveNew",
        "processing":
        [
            {
                "Step": 1,
                "method": "post",
                "end_point": "document",
                "map":
                [
                    {"ddate":"ddate"},
                    {"dnumber":"dnumber"},
                    {"note":"note"},
                    {"status":1},
                    {"id_user":1},
                    {"prefix":"prefix"},
                    {"id_transaction":"id_transaction"}
                ],
                "FK":
                [
                    {"id": "id"}
                ]
            },
            {
                "Step": 2,
                "method": "post",
                "end_point": "appointment_document",
                "fields":
                [
                    {"status": 1}
                ]
            }
        ]
    }',
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
