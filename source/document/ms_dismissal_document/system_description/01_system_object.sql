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
        { "field": "status", "label": "", "type": "status"},
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
        { "field": "dnumber", "readonly": true, "label": "Номер", "hidden": false, "required": true },
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
                    {"prefix":0},
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
                "end_point": "dismissal_document",
                "fields":
                [
                    {"status": 1}
                ]
            }
        ]
    }',
    '{
        "action": "onSaveEdit",
        "processing":
        [
            {
                "Step": 1,
                "method": "put",
                "end_point": "document",
                "filter": "id",
                "conditions": [
                    {"status": 1}
                ],
                "map":
                [
                    {"ddate":"ddate"},
                    {"dnumber":"dnumber"},
                    {"note":"note"},
                    {"status":1},
                    {"id_user":1},
                    {"prefix":0},
                    {"id_transaction":"id_transaction"}
                ]
            },
            {
                "Step": 2,
                "method": "put",
                "end_point": "dismissal_document",
                "fields":
                [
                    {"status": 1}
                ]
            }
        ]
    }',
    '{
    "action": "onAcceptance",
        "processing":
        [
            {
                "Step": 1,
                "method": "get",
                "hide_in_preview": true,
                "end_point": "dismissal_document",
                "filter": "id",
                "conditions":
                [
                    {"status": 1}
                ],
                "FK":
                [
                    {"id": "id"},
                    {"ddate": "ddate"},
                    {"dnumber": "dnumber"},
                    {"note": "note"},
                    {"id_employee": "id_employee"},
                    {"id_organization": "id_organization"},
                    {"id_department": "id_department"},
                    {"id_job_title": "id_job_title"},
                    {"justification": "justification"},
                    {"prefix": "prefix"},
                    {"id_transaction": "id_transaction"}
                ]
            },
            {
                "Step": 2,
                "method": "put",
                "hide_in_preview": true,
                "end_point": "document",
                "filter": "id",
                "map":
                [
                    {"ddate":"ddate"},
                    {"dnumber":"dnumber"},
                    {"note":"note"},
                    {"status":2},
                    {"id_user":1},
                    {"prefix":0},
                    {"id_transaction":"id_transaction"}
                ]
            },
            {
                "Step": 3,
                "method": "post",
                "end_point": "employee_data_ledger",
                "map": [
                    {"balance":          -1},
                    {"id_document":      "id"},
                    {"ddate":            "ddate"},
                    {"id_employee":      "id_employee"},
                    {"prefix":           "prefix"},
                    {"id_transaction":   "id_transaction"}
                ]
            },
            {
                "Step": 4,
                "method": "put",
                "hide_in_preview": true,
                "end_point": "dismissal_document",
                "filter": "id",
                "fields": {
                    "status": 2
                }
            }
        ]
    }',
    -- onDelete чистит оба регистра по id_document: employee_data — свои строки,
    -- staff_placement — только строки, ссылающиеся на этот приказ (наследие
    -- тестового сида, где увольнение писало и туда; новых circuit нет).
    '{
    "action":"onDelete",
    "processing":[
        {
            "step": 1,
            "method": "get",
            "hide_in_preview": true,
            "end_point": "dismissal_document",
            "filter": "id",
            "FK": [
                {"id_document": "id"}
            ]
        },
        {
            "step": 2,
            "method": "delete",
            "end_point": "staff_placement_ledger",
            "filter": "id_document"
        },
        {
            "step": 3,
            "method": "delete",
            "end_point": "employee_data_ledger",
            "filter": "id_document"
        },
        {
            "step": 4,
            "method": "put",
            "hide_in_preview": true,
            "end_point": "document",
            "filter": "id",
            "fields": {
                "status": 0
            }
        },
        {
            "step": 5,
            "method": "put",
            "hide_in_preview": true,
            "end_point": "dismissal_document",
            "filter": "id",
            "fields": {
                "status": 0
            }
        }
    ]
    }',
    '{
      "privileges": [
        { "open_list": "true" },
        { "open_item": "true" },
        { "create": "true" },
        { "edit": "true" },
        { "delete": "true" },
        { "accept": "true" },
        { "create_related_documents": "true" }
      ]
    }',
    '',
    '',
    ''
  );
