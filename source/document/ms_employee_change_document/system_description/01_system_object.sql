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
    'Изменение данных сотрудника',
    'Изменение ФИО сотрудника (пишет таймлайн ФИО в регистре, численность не меняет)',
    1,
    0,
    'ИД-',
    1,
    FALSE,
    2,
    'employee_change_document',
    5,
    'document.png',
    '{
      "columns": [
        { "field": "status", "label": "", "type": "status"},
        { "field": "ddate", "label": "Дата"},
        { "field": "dnumber", "label": "Номер"},
        { "field": "note", "label": "Примечание"},
        { "field": "id_employee", "label": "Сотрудник", "end_point": "employee_directory"},
        { "field": "last_name", "label": "Новая фамилия"},
        { "field": "first_name", "label": "Новое имя"},
        { "field": "middle_name", "label": "Новое отчество"}
      ]
    }',
    'Журнал изменения данных сотрудников',
    'Документ изменения данных сотрудника',
    '{
      "fields": [
        { "field": "id", "label": "Код", "hidden": true, "required": false },
        { "field": "ddate", "label": "Дата", "hidden": false, "required": true, "default": "now()" },
        { "field": "dnumber", "readonly": true, "label": "Номер", "hidden": false, "required": true },
        { "field": "note", "label": "Примечание", "hidden": false, "required": false },
        { "field": "id_employee", "label": "Сотрудник", "hidden": false, "required": true, "type": "select", "end_point": "employee_directory" },
        { "field": "last_name", "label": "Новая фамилия", "hidden": false, "required": true, "default": "" },
        { "field": "first_name", "label": "Новое имя", "hidden": false, "required": true, "default": "" },
        { "field": "middle_name", "label": "Новое отчество", "hidden": false, "required": true, "default": "" },
        { "field": "last_genitive", "label": "Фамилия (род.)", "hidden": false, "required": false, "default": "" },
        { "field": "first_genitive", "label": "Имя (род.)", "hidden": false, "required": false, "default": "" },
        { "field": "middle_genitive", "label": "Отчество (род.)", "hidden": false, "required": false, "default": "" },
        { "field": "last_accusative", "label": "Фамилия (вин.)", "hidden": false, "required": false, "default": "" },
        { "field": "first_accusative", "label": "Имя (вин.)", "hidden": false, "required": false, "default": "" },
        { "field": "middle_accusative", "label": "Отчество (вин.)", "hidden": false, "required": false, "default": "" },
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
                "end_point": "employee_change_document",
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
                "method": "put",
                "end_point": "employee_change_document",
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
                "end_point": "employee_change_document",
                "filter":"id",
                "conditions":
                [
                    {"status": 1}
                ]
            },
            {
                "Step": 2,
                "method": "post",
                "end_point": "employee_data_ledger",
                "map": [
                    {"delta":            0},
                    {"endpoint":         "employee_change_document"},
                    {"id_document":      "id"},
                    {"ddate":            "ddate"},
                    {"id_employee":      "id_employee"},
                    {"fio": {"CONCAT":
                        ["last_name", " ", "first_name", " ", "middle_name"]
                        }
                    }
                ]
            },
            {
                "Step": 3,
                "method": "put",
                "end_point": "employee_directory",
                "filter": {"id": "id_employee"},
                "map": [
                    {"name": {"CONCAT":
                        ["last_name", " ", "first_name", " ", "middle_name"]
                        }
                    }
                ]
            },
            {
                "Step": 4,
                "method": "put",
                "hide_in_preview": true,
                "end_point": "employee_change_document",
                "filter": "id",
                "fields": {
                    "status": 2
                }
            }
        ]
    }',
    -- onDelete чистит свою строку fio-таймлайна (пара id_document + endpoint).
    -- Имя в справочнике НЕ откатывается (крайний случай: таймлайн после удаления
    -- покажет предыдущее ФИО, справочник — последнее записанное).
    '{
    "action":"onDelete",
    "processing":[
        {
            "step": 1,
            "method": "get",
            "hide_in_preview": true,
            "end_point": "employee_change_document",
            "filter": "id",
            "FK": [
                {"id_document": "id"}
            ]
        },
        {
            "step": 2,
            "method": "delete",
            "end_point": "employee_data_ledger",
            "filter": "id_document",
            "query": {
                "endpoint": "employee_change_document"
            }
        },
        {
            "step": 3,
            "method": "put",
            "hide_in_preview": true,
            "end_point": "employee_change_document",
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
