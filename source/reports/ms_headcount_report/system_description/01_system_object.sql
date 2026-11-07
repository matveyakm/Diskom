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
    'Отчёт о численности сотрудников',
    'Отчёт о численности сотрудников за период',
    1,
    0,
    'ОЧ-',
    1,
    FALSE,
    3,
    'headcount_report',
    1,
    'document.png',
    '{
      "columns": [
        { "field": "status", "label": "", "type": "status", "hidden": false },
        { "field": "ddate", "label": "Дата", "hidden": false },
        { "field": "dnumber", "label": "Номер", "hidden": false },
        { "field": "date_from", "label": "Дата с", "hidden": false },
        { "field": "date_to", "label": "Дата по", "hidden": false }
      ]
    }',
    'Журнал отчётов о численности сотрудников',
    'Отчёт о численности сотрудников',
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
      "field": "date_from",
      "label": "Дата с",
      "hidden": false,
      "required": true,
      "type": "date"
    },
    {
      "field": "date_to",
      "label": "Дата по",
      "hidden": false,
      "required": true,
      "type": "date"
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
                "end_point": "headcount_report",
                "map":
                [
                    {"ddate": "ddate"},
                    {"dnumber": "dnumber"},
                    {"date_from": "date_from"},
                    {"date_to": "date_to"},
                    {"comment": "comment"},
                    {"status": 1},
                    {"id_user": 1},
                    {"prefix": 0},
                    {"id_transaction": "id_transaction"}
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
                "method": "put",
                "end_point": "headcount_report",
                "filter": "id",
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
    '{
    "action": "onAcceptance",
        "processing":
        [
            {
                "Step": 1,
                "method": "get",
                "hide_in_preview": true,
                "end_point": "headcount_report",
                "filter": "id",
                "conditions":
                [
                    {"status": 1}
                ]
            },
            {
                "Step": 2,
                "method": "put",
                "hide_in_preview": true,
                "end_point": "headcount_report",
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
            "end_point": "headcount_report",
            "filter": "id",
            "FK": [
                {"id_document": "id"}
            ]
        },
        {
            "step": 2,
            "method": "put",
            "hide_in_preview": true,
            "end_point": "headcount_report",
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
