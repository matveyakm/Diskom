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
    'Меры дисциплинарных взысканий',
    '',
    1,
    0,
    'МДО-',
    1,
    FALSE,
    1,
    'disciplinary_measures',
    10,
    'dictionary.png',
    '{
      "columns": [
        {
          "field": "id",
          "label": "",
          "hidden": true
        },
        {
          "field": "dnumber",
          "label": "Код",
          "hidden": false
        },
        {
          "field": "name",
          "label": "Наименование",
          "hidden": false
        },
        {
          "field": "description",
          "label": "Описание",
          "hidden": false
        }
      ]
    }',
    'Справочник мер дисциплинарных взысканий',
    'Меры дисциплинарных взысканий',
    '{
      "fields": [
        {
          "field": "id",
          "label": "",
          "hidden": true,
          "required": true
        },
        {
          "field": "dnumber",
          "label": "Код",
          "hidden": false,
          "required": false
        },
        {
          "field": "name",
          "label": "Наименование",
          "hidden": false,
          "required": false
        },
        {
          "field": "description",
          "label": "Описание",
          "hidden": false,
          "required": false
        }
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
