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
    'Должности',
    'Должности',
    1,
    0,
    'ДЛ-',
    1,
    FALSE,
    1,
    'job_title_directory',
    5,
    'dictionary.png',
    '{
      "columns": [
        { "field": "id", "label": "", "hidden": true},
        { "field": "status", "label": "", "type": "status", "hidden": false},
        { "field": "name", "label": "Название", "hidden": false},
        { "field": "description", "label": "Описание", "hidden": false}
      ]
    }',
    'Справочник должностей',
    'Элемент справочника должностей',
    '{
      "fields": [
        { "field": "id", "label": "", "hidden": true, "required": false },
        { "field": "name", "label": "Название (именительный)", "hidden": false, "required": true},
        { "field": "name_genitive", "label": "Родительный (кого? чего?)", "hidden": false, "required": false},
        { "field": "name_dative", "label": "Дательный (кому? чему?)", "hidden": false, "required": false},
        { "field": "name_accusative", "label": "Винительный (кого? что?)", "hidden": false, "required": false},
        { "field": "name_instrumental", "label": "Творительный (кем? чем?)", "hidden": false, "required": false},
        { "field": "name_prepositional", "label": "Предложный (о ком? о чём?)", "hidden": false, "required": false},
        { "field": "description", "label": "Описание", "hidden": false, "required": false}
      ]
    }',
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
