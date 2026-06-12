import json
import os
import traceback
from pathlib import Path
from flask import Flask, render_template, request, redirect, url_for, flash, jsonify

from project_reader import (
    read_all_microservices,
    read_system_object_entities,
    build_entity_index,
    SOURCE_DIR,
)
from entity_scaffolder import (
    generate_main_go,
    generate_go_mod,
    generate_parameters_env,
    generate_model_go,
    generate_db_sh,
    generate_dockerfile,
    generate_system_object_insert,
    append_insert_to_db_sh,
    update_all_dockerfiles,
    get_services_grouped,
    get_db_user_from_ref,
    get_ms_name_from_ref,
    generate_table_columns_json,
    generate_item_fields_json,
)
from docker_compose_editor import (
    get_next_ips,
    add_microservice_to_compose,
    add_to_go_work,
    get_compose_content,
)

app = Flask(__name__)
app.secret_key = "dev-tool-secret-key"

BASE_FIELDS_BY_CATEGORY = {
    "directory": [
        {"name": "Id", "go_type": "int64", "column": "id", "json_name": "id",
         "is_pk": True, "is_auto_increment": True, "not_null": True,
         "hidden": True},
        {"name": "DNumber", "go_type": "string", "column": "dnumber", "json_name": "dnumber",
         "ui_label": "Код", "is_pk": False, "is_auto_increment": False, "not_null": False},
        {"name": "Name", "go_type": "string", "column": "name", "json_name": "name",
         "ui_label": "Наименование", "is_pk": False, "is_auto_increment": False, "not_null": False},
        {"name": "Description", "go_type": "string", "column": "description", "json_name": "description",
         "ui_label": "Описание", "is_pk": False, "is_auto_increment": False, "not_null": False},
        {"name": "Status", "go_type": "int16", "column": "status", "json_name": "status",
         "ui_label": "Статус", "is_pk": False, "is_auto_increment": False, "not_null": False,
         "exclude_from_form": True},
        {"name": "IdBasis", "go_type": "int64", "column": "id_basis", "json_name": "id_basis",
         "ui_label": "", "is_pk": False, "is_auto_increment": False, "not_null": False,
         "hidden": True, "exclude_from_table": True, "exclude_from_form": True},
        {"name": "Prefix", "go_type": "int16", "column": "prefix", "json_name": "prefix",
         "ui_label": "", "is_pk": False, "is_auto_increment": False, "not_null": False,
         "hidden": True, "exclude_from_table": True, "exclude_from_form": True},
        {"name": "IdTransaction", "go_type": "string", "column": "id_transaction",
         "json_name": "id_transaction", "ui_label": "", "is_pk": False,
         "is_auto_increment": False, "not_null": False,
         "hidden": True, "exclude_from_table": True, "exclude_from_form": True},
        {"name": "IdUser", "go_type": "int64", "column": "id_user", "json_name": "id_user",
         "ui_label": "", "is_pk": False, "is_auto_increment": False, "not_null": False,
         "hidden": True, "exclude_from_table": True, "exclude_from_form": True},
    ],
    "document": [
        {"name": "Id", "go_type": "int64", "column": "id", "json_name": "id",
         "is_pk": True, "is_auto_increment": False, "not_null": True,
         "hidden": True},
        {"name": "Ddate", "go_type": "time.Time", "column": "ddate", "json_name": "ddate",
         "ui_label": "Дата", "is_pk": False, "is_auto_increment": False, "not_null": False},
        {"name": "DNumber", "go_type": "string", "column": "dnumber", "json_name": "dnumber",
         "ui_label": "Номер", "is_pk": False, "is_auto_increment": False, "not_null": False},
        {"name": "Note", "go_type": "string", "column": "note", "json_name": "note",
         "ui_label": "Примечание", "is_pk": False, "is_auto_increment": False, "not_null": False},
        {"name": "IdBasis", "go_type": "int64", "column": "id_basis", "json_name": "id_basis",
         "ui_label": "", "is_pk": False, "is_auto_increment": False, "not_null": False,
         "hidden": True, "exclude_from_table": True, "exclude_from_form": True},
        {"name": "Status", "go_type": "int16", "column": "status", "json_name": "status",
         "ui_label": "Статус", "is_pk": False, "is_auto_increment": False, "not_null": False,
         "exclude_from_form": True},
        {"name": "IdUser", "go_type": "int64", "column": "id_user", "json_name": "id_user",
         "ui_label": "", "is_pk": False, "is_auto_increment": False, "not_null": False,
         "hidden": True, "exclude_from_table": True, "exclude_from_form": True},
        {"name": "Prefix", "go_type": "int16", "column": "prefix", "json_name": "prefix",
         "ui_label": "", "is_pk": False, "is_auto_increment": False, "not_null": False,
         "hidden": True, "exclude_from_table": True, "exclude_from_form": True},
        {"name": "IdTransaction", "go_type": "string", "column": "id_transaction",
         "json_name": "id_transaction", "ui_label": "", "is_pk": False,
         "is_auto_increment": False, "not_null": False,
         "hidden": True, "exclude_from_table": True, "exclude_from_form": True},
    ],
    "ledger": [
        {"name": "Id", "go_type": "int64", "column": "id", "json_name": "id",
         "is_pk": True, "is_auto_increment": True, "not_null": True,
         "hidden": True},
        {"name": "IdDocument", "go_type": "int64", "column": "id_document",
         "json_name": "id_document", "ui_label": "Документ", "is_pk": False,
         "is_auto_increment": False, "not_null": True},
        {"name": "Ddate", "go_type": "time.Time", "column": "ddate", "json_name": "ddate",
         "ui_label": "Дата", "is_pk": False, "is_auto_increment": False, "not_null": False},
        {"name": "Prefix", "go_type": "int16", "column": "prefix", "json_name": "prefix",
         "ui_label": "", "is_pk": False, "is_auto_increment": False, "not_null": False,
         "hidden": True, "exclude_from_table": True, "exclude_from_form": True},
        {"name": "IdTransaction", "go_type": "string", "column": "id_transaction",
         "json_name": "id_transaction", "ui_label": "", "is_pk": False,
         "is_auto_increment": False, "not_null": False,
         "hidden": True, "exclude_from_table": True, "exclude_from_form": True},
    ],
}

GROUP_NAMES = {1: "Справочники", 2: "Документы", 3: "Отчеты", 4: "Регистры данных"}


@app.template_filter("tojson_pretty")
def tojson_pretty(value):
    if isinstance(value, (dict, list)):
        return json.dumps(value, ensure_ascii=False, indent=2)
    if isinstance(value, str) and value.strip().startswith(("{", "[")):
        try:
            return json.dumps(json.loads(value), ensure_ascii=False, indent=2)
        except json.JSONDecodeError:
            pass
    return str(value)


@app.route("/")
def index():
    index_data = build_entity_index()
    entities = []

    for ref, data in sorted(index_data.items()):
        seed = data.get("seed", {}) or {}
        ms = data.get("ms")
        entities.append({
            "ref": ref,
            "name": seed.get("name", ref),
            "description": seed.get("description", ""),
            "is_group": seed.get("is_group", False),
            "id_group": seed.get("id_group", 0),
            "group_name": GROUP_NAMES.get(seed.get("id_group", 0), ""),
            "has_seed": bool(seed.get("name")),
            "has_ms": ms is not None,
            "category": ms["category"] if ms else "",
            "table_name": ms["table_name"] if ms else ref,
            "db_host": ms["db_host"] if ms else "",
            "fields_count": len(ms["fields"]) if ms and ms.get("fields") else 0,
        })

    ms_list = read_all_microservices()
    seed_list = read_system_object_entities()

    return render_template(
        "index.html",
        entities=entities,
        ms_count=len(ms_list),
        seed_count=len(seed_list),
        group_names=GROUP_NAMES,
    )


@app.route("/add", methods=["GET", "POST"])
def add_entity():
    if request.method == "POST":
        try:
            data = request.form
            ref = data.get("ref", "").strip().lower().replace(" ", "_")
            category = data.get("category", "directory")
            name = data.get("name", ref).strip()
            description = data.get("description", "").strip()
            id_group = int(data.get("id_group", 1))
            output_order = int(data.get("output_order", 10))
            prefix = int(data.get("prefix", 0))
            prefix_number = data.get("prefix_number", "")
            icon = data.get("icon", "dictionary.png")
            table_title = data.get("table_title", "")
            item_title = data.get("item_title", "")
            level = int(data.get("level", 1))

            if not ref:
                flash("ref (endpoint name) is required", "error")
                return redirect(url_for("add_entity"))

            fields_json = data.get("fields_json", "[]")
            all_fields = enrich_fields(json.loads(fields_json))

            ms_name = get_ms_name_from_ref(ref)
            db_user = get_db_user_from_ref(ref)
            db_ips = get_next_ips()
            db_host = db_ips[0]

            ms_dir = SOURCE_DIR / category / ms_name
            model_dir = ms_dir / "model"
            dbshell_dir = ms_dir / "dbshell"
            model_dir.mkdir(parents=True, exist_ok=True)
            dbshell_dir.mkdir(parents=True, exist_ok=True)

            (ms_dir / "go.mod").write_text(generate_go_mod(ms_name))
            (ms_dir / "main.go").write_text(generate_main_go(ms_name))
            (ms_dir / "parameters.env").write_text(
                generate_parameters_env(ref, db_host, db_user, ref)
            )
            (model_dir / "model.go").write_text(generate_model_go(all_fields, ref))
            (dbshell_dir / "db.sh").write_text(generate_db_sh(ref, all_fields, category, db_user))
            (dbshell_dir / "db.sh").chmod(0o755)

            all_services = get_services_grouped()
            dockerfile_dir = Path(__file__).parent.parent.parent / "Dockerfiles"
            dockerfile_dir.mkdir(exist_ok=True)
            (dockerfile_dir / ms_name).write_text(
                generate_dockerfile(category, ms_name, all_services)
            )

            # Add new service to ALL existing Dockerfiles
            updated_dfs = update_all_dockerfiles(ref, category)

            comp_ok, comp_msg = add_microservice_to_compose(ref, category)
            gow_ok, gow_msg = add_to_go_work(ref, category)

            # Generate INSERT for system_object_directory and write to per-entity file
            insert_sql = generate_system_object_insert(
                name=name, description=description, ref=ref,
                prefix=prefix, prefix_number=prefix_number,
                level=level, is_group=False, id_group=id_group,
                output_order=output_order, icon=icon,
                table_title=table_title, item_title=item_title,
                fields=all_fields, category=category,
            )
            seed_ok, seed_msg = append_insert_to_db_sh(insert_sql, ref=ref, category=category)

            df_msg = f"Dockerfiles updated: {len(updated_dfs)}" if updated_dfs else ""
            flash(
                f"Entity '{ref}' created: {comp_msg}; {gow_msg}; {seed_msg}; {df_msg}",
                "success",
            )
            return redirect(url_for("index"))

        except Exception as e:
            flash(f"Error: {e}<br><pre>{traceback.format_exc()}</pre>", "error")
            return redirect(url_for("add_entity"))

    return render_template(
        "add.html",
        categories=["directory", "document", "ledger"],
        base_fields_json=json.dumps(BASE_FIELDS_BY_CATEGORY, ensure_ascii=False),
        group_names=GROUP_NAMES,
        next_ips=get_next_ips(),
    )


@app.route("/detail/<ref>")
def entity_detail(ref):
    index_data = build_entity_index()
    data = index_data.get(ref)
    if not data:
        flash(f"Entity '{ref}' not found", "error")
        return redirect(url_for("index"))

    seed = data.get("seed", {}) or {}
    ms = data.get("ms")

    preview = {}
    if ms:
        category = ms["category"]
        ms_name = ms["name"]
        all_fields = ms.get("fields", []) or []
        preview["main_go"] = generate_main_go(ms_name)
        preview["go_mod"] = generate_go_mod(ms_name)
        preview["parameters_env"] = generate_parameters_env(
            ref, ms.get("db_host", ""), ms.get("db_user", ""), ms.get("table_name", ref)
        )
        preview["model_go"] = generate_model_go(all_fields, ref)
        preview["db_sh"] = generate_db_sh(
            ms.get("table_name", ref), all_fields, category, ms.get("db_user", "")
        )
        all_services = get_services_grouped()
        preview["dockerfile"] = generate_dockerfile(category, ms_name, all_services)
        preview["insert_sql"] = generate_system_object_insert(
            name=seed.get("name", ref),
            description=seed.get("description", ""),
            ref=ref,
            prefix=seed.get("prefix", 0),
            prefix_number=seed.get("prefix_number", ""),
            level=seed.get("level", 1),
            is_group=seed.get("is_group", False),
            id_group=seed.get("id_group", 1),
            output_order=seed.get("output_order", 10),
            icon=seed.get("icon", "dictionary.png"),
            table_title=seed.get("table_title", ""),
            item_title=seed.get("item_title", ""),
            fields=all_fields, category=category,
        )

    return render_template(
        "detail.html",
        ref=ref,
        seed=seed,
        ms=ms,
        preview=preview,
        group_names=GROUP_NAMES,
    )


SQL_TYPE_MAP = {
    "int64": "bigint", "int16": "smallint", "string": "text",
    "time.Time": "timestamp", "bool": "boolean", "float64": "double precision",
    "int": "bigint",
}


def enrich_fields(fields):
    """Add sql_type and default column/json name to fields."""
    result = []
    for f in fields:
        f = dict(f)
        if "sql_type" not in f or not f["sql_type"]:
            f["sql_type"] = SQL_TYPE_MAP.get(f.get("go_type", "string"), "text")
        if not f.get("column"):
            f["column"] = f.get("json_name", f.get("name", "unknown")).lower()
        if not f.get("json_name"):
            f["json_name"] = f.get("column", f.get("name", "unknown"))
        result.append(f)
    return result


@app.route("/preview-files", methods=["POST"])
def preview_files():
    data = request.json
    ref = data.get("ref", "").strip().lower().replace(" ", "_")
    category = data.get("category", "directory")
    fields = data.get("fields", [])

    all_fields = enrich_fields(fields)

    ms_name = get_ms_name_from_ref(ref)
    db_user = get_db_user_from_ref(ref)
    db_host = get_next_ips()[0]

    all_services = get_services_grouped()
    return jsonify({
        "main_go": generate_main_go(ms_name),
        "go_mod": generate_go_mod(ms_name),
        "parameters_env": generate_parameters_env(ref, db_host, db_user, ref),
        "model_go": generate_model_go(all_fields, ref),
        "db_sh": generate_db_sh(ref, all_fields, category, db_user),
        "dockerfile": generate_dockerfile(category, ms_name, all_services),
        "table_columns": generate_table_columns_json(all_fields),
        "item_fields": generate_item_fields_json(all_fields, category),
        "insert_sql": generate_system_object_insert(
            name=ref.replace("_", " ").title(), description="",
            ref=ref, prefix=0, prefix_number="",
            level=1, is_group=False, id_group=1,
            output_order=10, icon="dictionary.png",
            table_title="", item_title="", fields=all_fields,
            category=category,
        ),
        "ms_name": ms_name,
        "db_host": db_host,
        "db_user": db_user,
    })


if __name__ == "__main__":
    print("=" * 60)
    print(" Entity Manager - Dev Tool for Disciplinary Committee")
    print("=" * 60)
    print(f" Project root: {SOURCE_DIR.parent}")
    print(f" Open: http://localhost:5234")
    print("=" * 60)
    app.run(debug=True, host="0.0.0.0", port=5234)
