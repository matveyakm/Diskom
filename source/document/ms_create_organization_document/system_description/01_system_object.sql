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
    'Регистрация организации/структуры',
    'Регистрация организации/структуры',
    1,
    0,
    'ПО-',
    1,
    FALSE,
    2,
    'create_organization_document',
    1,
    'document.png',
    '{
      "columns": [
        { "field": "status", "label": "", "type": "status", "hidden": false },
        { "field": "ddate", "label": "Дата", "hidden": false },
        { "field": "dnumber", "label": "Номер", "hidden": false },
        { "field": "note", "label": "Примечание", "hidden": false },
        { "field": "name", "label": "Название", "hidden": false }
      ]
    }',
    'Журнал документов создание организации/структуры',
    'Документ создания организации/структуры',
    '{
      "fields": [
        { "field": "id", "label": "", "hidden": true, "required": false },
        { "field": "ddate", "label": "Дата", "hidden": false, "required": true, "default": "now()" },
        { "field": "dnumber", "readonly": true, "label": "Номер", "hidden": false, "required": true },
        { "field": "name", "label": "Название", "hidden": false, "required": false, "default": "" },
        { "field": "note", "label": "Примечание", "hidden": false, "required": false, "default": "" },
        { "field": "id_user", "label": "", "hidden": true, "required": false, "default": 1 },
        { "field": "id_basis", "label": "", "hidden": true, "required": false, "default": 0 },
        { "field": "prefix", "label": "", "hidden": true, "required": false, "default": 0 }
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
                "end_point": "create_organization_document",
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
                    {"name":"name"},
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
                "method": "put",
                "end_point": "create_organization_document",
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
                "end_point": "create_organization_document",
                "filter":"id",
                "conditions":
                [
                    {"status": 1}
                ],
                "FK":
                [
                    {"id": "id"},
                    {"ddate":"ddate"},
                    {"dnumber":"dnumber"},
                    {"name":"name"},
                    {"note":"note"},
                    {"prefix":"prefix"},
                    {"id_transaction":"id_transaction"}
                ]
            },
            {
                "Step": 2,
                "method": "put",
                "hide_in_preview": true,
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
                "end_point": "organization_directory",
                "map": [
                    {"description": "note"},
                    {"prefix":"prefix"},
                    {"status": 1},
                    {"id_basis": "id"},
                    {"name": "name"}
                ],
                "FK": [
                    {"id_organization": "id"}
                ]
            },
            {
                "Step": 4,
                "method": "post",
                "end_point": "organization_data_ledger",
                "map": [
                    {"balance":          1},
                    {"id_document":      "id"},
                    {"ddate":            "ddate"},
                    {"id_organization":  "id_organization"},
                    {"name":             "name"},
                    {"prefix":           "prefix"},
                    {"id_transaction":   "id_transaction"}
                ]
            },
            {
                "Step": 5,
                "method": "put",
                "hide_in_preview": true,
                "end_point": "create_organization_document",
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
            "end_point": "create_organization_document",
            "filter": "id"
        },
        {
            "step": 2,
            "method": "delete",
            "end_point": "organization_data_ledger",
            "map": [
                {"id_document": "id"}
            ]
        },
        {
            "step": 3,
            "method": "put",
            "end_point": "organization_directory",
            "filter": "id_organization",
            "fields": {
                "status": 0
            }
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
            "end_point": "create_organization_document",
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
