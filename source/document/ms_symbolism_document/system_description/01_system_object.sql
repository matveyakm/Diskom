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
    'Приказ об установлении символики',
    'Приказ об установлении символики',
    1,
    0,
    'СИ-',
    1,
    FALSE,
    2,
    'symbolism_document',
    4,
    'document.png',
    '{
      "columns": [
        { "field": "status", "label": "", "type": "status", "hidden": false },
        { "field": "ddate", "label": "Дата", "hidden": false },
        { "field": "dnumber", "label": "Номер", "hidden": false },
        { "field": "id_committee", "label": "Комитет", "hidden": false, "end_point": "department_directory" },
        { "field": "id_logo", "label": "Логотип", "hidden": false, "type": "file" }
      ]
    }',
    'Журнал приказов об установлении символики',
    'Приказ об установлении символики',
    '{
  "fields": [
    {
      "field": "id",
      "label": "",
      "hidden": true,
      "required": false
    },
    {
      "field": "ddate",
      "label": "Дата",
      "hidden": false,
      "required": true,
      "default": "now()"
    },
    {
      "field": "dnumber",
      "readonly": true,
      "label": "Номер",
      "hidden": false,
      "required": true
    },
    {
      "field": "id_committee",
      "label": "Комитет",
      "hidden": false,
      "required": true,
      "type": "select",
      "end_point": "department_directory"
    },
    {
      "field": "id_logo",
      "label": "Логотип",
      "hidden": false,
      "required": true,
      "type": "file"
    },
    {
      "field": "comment",
      "label": "Комментарий",
      "hidden": false,
      "required": false,
      "type": "textarea",
      "default": ""
    },
    {
      "field": "id_user",
      "label": "",
      "hidden": true,
      "required": false,
      "default": 1
    },
    {
      "field": "id_basis",
      "label": "",
      "hidden": true,
      "required": false,
      "default": 0
    },
    {
      "field": "prefix",
      "label": "",
      "hidden": true,
      "required": false,
      "default": 0
    }
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
                "end_point": "symbolism_document",
                "fields":
                [
                    {"status": 1}
                ]
            },
            {
                "Step": 3,
                "method": "put",
                "end_point": "symbolism_document/files",
                "map":
                [
                    {"id": "id_logo"},
                    {"id_document": "id"},
                    {"replace": true}
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
                "end_point": "symbolism_document",
                "fields":
                [
                    {"status": 1}
                ]
            },
            {
                "Step": 3,
                "method": "put",
                "end_point": "symbolism_document/files",
                "map":
                [
                    {"id": "id_logo"},
                    {"id_document": "id"},
                    {"replace": true}
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
                "end_point": "symbolism_document",
                "filter": "id",
                "conditions":
                [
                    {"status": 1}
                ],
                "FK":
                [
                    {"id": "id"},
                    {"ddate":"ddate"},
                    {"dnumber":"dnumber"},
                    {"id_committee":"id_committee"},
                    {"id_logo":"id_logo"},
                    {"comment":"comment"},
                    {"prefix":"prefix"},
                    {"id_transaction":"id_transaction"}
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
                    {"status":2},
                    {"id_user":1},
                    {"prefix":"prefix"},
                    {"id_transaction":"id_transaction"}
                ]
            },
            {
                "Step": 3,
                "method": "post",
                "end_point": "symbolism_data_ledger",
                "map": [
                    {"id_document":      "id"},
                    {"ddate":            "ddate"},
                    {"id_committee":     "id_committee"},
                    {"id_logo":          "id_logo"},
                    {"comment":          "comment"},
                    {"prefix":           "prefix"},
                    {"id_transaction":   "id_transaction"}
                ]
            },
            {
                "Step": 4,
                "method": "put",
                "hide_in_preview": true,
                "end_point": "symbolism_document",
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
            "hide_in_preview": true,
            "end_point": "symbolism_document",
            "filter": "id",
            "FK": [
                {"id_document": "id"}
            ]
        },
        {
            "step": 2,
            "method": "delete",
            "end_point": "symbolism_document/files",
            "filter": "id_document"
        },
        {
            "step": 3,
            "method": "delete",
            "end_point": "symbolism_data_ledger",
            "map": [
                {"id_document": "id"}
            ]
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
            "end_point": "symbolism_document",
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
        { "create_related_documents": "false" }
      ]
    }',
    '',
    '',
    ''
  );
