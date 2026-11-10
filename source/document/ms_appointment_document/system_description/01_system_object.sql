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
        { "field": "status", "label": "", "type": "status"},
        { "field": "ddate", "label": "Дата"},
        { "field": "dnumber", "label": "Номер"},
        { "field": "note", "label": "Примечание"},
        { "field": "id_employee", "label": "Сотрудник", "end_point": "employee_directory"},
        { "field": "id_organization", "label": "Комитет", "end_point": "organization_directory"},
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
                "end_point": "appointment_document",
                "fields":
                [
                    {"status": 1}
                ],
                "FK":
                [
                    {"id": "id"}
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
                "method": "post",
                "end_point": "appointment_document",
                "conditions": [
                    {"status": 1}
                ],
                "fields":
                [
                    {"status": 1}
                ]
            }
        ]
    }',
    -- Превью confirm: шаги с "hide_in_preview": true скрываются (контракт: frontend/src/shared/api/actionPipeline/effects.ts).
    '{
    "action": "onAcceptance",
        "processing":
        [
            {
                "Step": 1,
                "method": "get",
                "hide_in_preview": true,
                "end_point": "appointment_document",
                "filter":"id",
                "conditions":
                [
                    {"status": 1}
                ]
            },
            {
                "Step": 2,
                "method": "get",
                "hide_in_preview": true,
                "end_point": "employee_directory",
                "filter": {"id": "id_employee"},
                "merge": false,
                "FK":
                [
                    {"employee_name": "name"}
                ]
            },
            {
                "Step": 3,
                "method": "get",
                "hide_in_preview": true,
                "end_point": "job_title_directory",
                "filter": {"id": "id_job_title"},
                "merge": false,
                "FK":
                [
                    {"job_title_name": "name"},
                    {"job_title_instrumental": "name_instrumental"}
                ]
            },
            {
                "Step": 4,
                "method": "get",
                "hide_in_preview": true,
                "end_point": "department_directory",
                "filter": {"id": "id_department"},
                "merge": false,
                "FK":
                [
                    {"department_name": "name"},
                    {"department_genitive": "name_genitive"}
                ]
            },
            {
                "Step": 5,
                "method": "post",
                "end_point": "staff_placement_ledger",
                "map": [
                    {"endpoint":               "appointment_document"},
                    {"event":                  "open"},
                    {"id_document":            "id"},
                    {"ddate":                  "ddate"},
                    {"id_employee":            "id_employee"},
                    {"employee_name":          "employee_name"},
                    {"id_organization":        "id_organization"},
                    {"id_department":          "id_department"},
                    {"department_name":        "department_name"},
                    {"department_genitive":    "department_genitive"},
                    {"id_job_title":           "id_job_title"},
                    {"job_title_name":         "job_title_name"},
                    {"job_title_instrumental": "job_title_instrumental"},
                    {"justification":          "justification"},
                    {"prefix":                 "prefix"},
                    {"id_transaction":         "id_transaction"}
                ]
            },
            {
                "Step": 6,
                "method": "put",
                "hide_in_preview": true,
                "end_point": "appointment_document",
                "filter": "id",
                "fields": {
                    "status": 2
                }
            }
        ]
    }',
    -- Превью confirm: шаги с "hide_in_preview": true скрываются (контракт: frontend/src/shared/api/actionPipeline/effects.ts).
    '{
    "action":"onDelete",
    "processing":[
        {
            "step": 1,
            "method": "get",
            "hide_in_preview": true,
            "end_point": "appointment_document",
            "filter": "id"
        },
        {
            "step": 2,
            "method": "delete",
            "end_point": "staff_placement_ledger",
            "filter": "id_document",
            "query": {
                "endpoint": "appointment_document"
            }
        },
        {
            "step": 3,
            "method": "put",
            "hide_in_preview": true,
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
