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
    'Назначение на должность',
    'Назначение на должность',
    1,
    0,
    'НАД-',
    1,
    FALSE,
    2,
    'appointment_document',
    2,
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
    'Журнал документов назначения на должность',
    'Документ назначения на должность',
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
                "end_point": "appointment_document",
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
    '{
    "action": "onAcceptance",
        "processing":
        [
            {
                "Step": 1,
                "method": "get",
                "end_point": "appointment_document",
                "filter":"id",
                "conditions":
                [
                    {"status": 1}
                ]
            },
            {
                "Step": 2,
                "method": "put",
                "end_point": "document",
                "filter":"id",
                "map":
                [
                    {"ddate":"ddate"},
                    {"dnumber":"dnumber"},
                    {"note":"note"},
                    {"status":2},
                    {"id_user":1},
                    {"prefix":"prefix"},
                    {"id_transaction":"id_transaction"}
                ]
            },
            {
                "Step": 3,
                "method": "post",
                "end_point": "staff_placement_ledger",
                "map": [
                    {"balance":           1},
                    {"id_document":      "id"},
                    {"ddate":            "ddate"},
                    {"id_employee":      "id_employee"},
                    {"id_organization":  "id_organization"},
                    {"id_department":    "id_department"},
                    {"id_job_title":     "id_job_title"},
                    {"id_jastification": "id_jastification"},
                    {"prefix":           "prefix"},
                    {"id_transaction":   "id_transaction"}
                ]
            },
            {
                "Step": 4,
                "method": "put",
                "end_point": "appointment_document",
                "filter": "id",
                "fields": {
                    "status": 2
                }
            }
        ]
    }',
    '{
    "action":"onDelete",
    "processing":[
        {
            "step": 1,
            "method": "get",
            "end_point": "appointment_document",
            "filter": "id"
        },
        {
            "step": 2,
            "method": "delete",
            "end_point": "staff_placement_ledger",
            "filter": "id_document"
        },
        {
            "step": 4,
            "method": "put",
            "end_point": "document",
            "filter": "id",
            "fields": {
                "status": 0
            }
        },
        {
            "step": 5,
            "method": "put",
            "end_point": "appointment_document",
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
