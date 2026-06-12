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
    'Регистрация обращения',
    'Регистрация обращения',
    1,
    0,
    'РО-',
    1,
    FALSE,
    2,
    'create_counterparty_document',
    1,
    'document.png',
    '{
      "columns": [
        { "field": "id", "label": "", "hidden": true },
        { "field": "status", "label": "", "hidden": false },
        { "field": "ddate", "label": "Дата", "hidden": false },
        { "field": "dnumber", "label": "Номер", "hidden": false },
        { "field": "note", "label": "Примечание", "hidden": false },
        { "field": "last_name", "label": "Фамилия", "hidden": false },
        { "field": "first_name", "label": "Имя", "hidden": false },
        { "field": "middle_name", "label": "Отчество", "hidden": false }
      ]
    }',
    'Журнал документов регистрация обращения',
    'Документ регистрация обращения',
    '{
      "fields": [
        { "field": "id", "label": "", "hidden": true, "required": false },
        { "field": "ddate", "label": "Дата", "hidden": false, "required": true, "default": "now()" },
        { "field": "dnumber", "label": "Номер", "hidden": false, "required": true, "default": "id" },
        { "field": "note", "label": "Примечание", "hidden": false, "required": false, "default": "" },
        { "field": "last_name", "label": "Фамилия", "hidden": false, "required": false, "default": "" },
        { "field": "first_name", "label": "Имя", "hidden": false, "required": false, "default": "" },
        { "field": "middle_name", "label": "Отчество", "hidden": false, "required": false, "default": "" },
        { "field": "last_genitive", "label": "Фамилия(Р)", "hidden": false, "required": false, "default": "" },
        { "field": "first_genitive", "label": "Имя(Р)", "hidden": false, "required": false, "default": "" },
        { "field": "middle_genitive", "label": "Отчество(Р)", "hidden": false, "required": false, "default": "" },
        { "field": "last_accusative", "label": "Фамилия(В)", "hidden": false, "required": false, "default": "" },
        { "field": "first_accusative", "label": "Имя(В)", "hidden": false, "required": false, "default": "" },
        { "field": "middle_accusative", "label": "Отчество(В)", "hidden": false, "required": false, "default": "" },
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
                    {"id_basis":"id_basis"},
                    {"id_user":"id_user"},
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
                "end_point": "create_counterparty_document",
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
                "method": "put",
                "end_point": "create_counterparty_document",
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
                "end_point": "create_counterparty_document",
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
                    {"note":"note"},
                    {"id_vk":"id_vk"},
                    {"prefix":"prefix"},
                    {"id_transaction":"id_transaction"}
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
                "end_point": "counterparty_directory",
                "map": [
                    {"description": "note"},
                    {"prefix":"prefix"},
                    {"status": 1},
                    {"id_basis": "id"},
                    {"name": {"CONCAT":
                        ["last_name", " ", "first_name", " ", "middle_name"]
                        }
                    }
                ],
                "FK": [
                    {"id_counterparty": "id"}
                ]
            },
            {
                "Step": 4,
                "method": "post",
                "end_point": "counterparty_data_ledger",
                "map": [
                    {"balance":          1},
                    {"id_document":      "id"},
                    {"ddate":            "ddate"},
                    {"id_counterparty":  "id_counterparty"},
                    {"first_name":       "first_name"},
                    {"last_name":        "last_name"},
                    {"middle_name":      "middle_name"},
                    {"first_genitive":   "first_genitive"},
                    {"last_genitive":    "last_genitive"},
                    {"middle_genitive":  "middle_genitive"},
                    {"first_accusative": "first_accusative"},
                    {"last_accusative":  "last_accusative"},
                    {"middle_accusative":"middle_accusative"},
                    {"prefix":           "prefix"},
                    {"id_transaction":   "id_transaction"}
                ]
            },
            {
                "Step": 5,
                "method": "put",
                "end_point": "create_counterparty_document",
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
            "end_point": "create_counterparty_document",
            "filter": "id"
        },
        {
            "step": 2,
            "method": "delete",
            "end_point": "counterparty_data_ledger",
            "filter": "id_document"
        },
        {
            "step": 3,
            "method": "put",
            "end_point": "counterparty_directory",
            "filter": "id_counterparty",
            "fields": {
                "status": 0
            }
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
            "end_point": "create_counterparty_document",
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
