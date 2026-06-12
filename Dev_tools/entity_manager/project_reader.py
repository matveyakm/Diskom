import re
import os
from pathlib import Path

SOURCE_DIR = Path(__file__).parent.parent.parent / "source"


def read_all_microservices():
    """Parse go.work and all model/model.go files to build a microservice list."""
    services = []
    go_work = SOURCE_DIR / "go.work"

    if not go_work.exists():
        return services

    content = go_work.read_text()
    use_block = re.search(r"use\s*\((.*?)\)", content, re.DOTALL)
    if not use_block:
        return services

    for line in use_block.group(1).strip().splitlines():
        line = line.strip().strip("./")
        if not line or line == "shared":
            continue
        parts = line.split("/")
        if len(parts) != 2:
            continue
        category, name = parts
        ms_ref = name.removeprefix("ms_")
        table_name = ms_ref

        model_path = SOURCE_DIR / category / name / "model" / "model.go"
        params_path = SOURCE_DIR / category / name / "parameters.env"

        fields = parse_gorm_model(model_path) if model_path.exists() else []
        db_host = ""
        db_user = ""
        if params_path.exists():
            for env_line in params_path.read_text().splitlines():
                if env_line.startswith("DB_HOST="):
                    db_host = env_line.split("=", 1)[1]
                elif env_line.startswith("DB_USER="):
                    db_user = env_line.split("=", 1)[1]
                elif env_line.startswith("DB_TABLENAME="):
                    table_name = env_line.split("=", 1)[1]

        services.append({
            "category": category,
            "name": name,
            "ref": ms_ref,
            "table_name": table_name,
            "exists_on_disk": True,
            "db_host": db_host,
            "db_user": db_user,
            "fields": fields,
        })
    return services


GO_TYPE_SQL_MAP = {
    "int64": "bigint",
    "int16": "smallint",
    "int": "bigint",
    "string": "text",
    "time.Time": "timestamp",
    "bool": "boolean",
    "float64": "double precision",
}

SQL_TYPE_GO_MAP = {v: k for k, v in GO_TYPE_SQL_MAP.items()}


def parse_gorm_model(filepath):
    """Parse a model.go file to extract field definitions."""
    content = Path(filepath).read_text()
    struct_match = re.search(r"type\s+Object\s+struct\s*\{", content)
    if not struct_match:
        return []

    start = struct_match.end()
    brace_depth = 1
    i = start
    while i < len(content) and brace_depth > 0:
        if content[i] == "{":
            brace_depth += 1
        elif content[i] == "}":
            brace_depth -= 1
        i += 1

    struct_body = content[start : i - 1]
    fields = []

    for line in struct_body.splitlines():
        line = line.strip()
        if not line or line.startswith("//"):
            continue

        m = re.match(r"(\w+)\s+([\w.]+)\s+`(.+?)`", line)
        if not m:
            continue

        name, go_type, tags_str = m.groups()

        gorm_tag = ""
        json_tag = ""
        for tag_part in tags_str.split():
            if tag_part.startswith("gorm:"):
                gorm_tag = tag_part[5:].strip('"')
            elif tag_part.startswith("json:"):
                json_tag = tag_part[5:].strip('"')

        gorm_attrs = {}
        if gorm_tag:
            for attr in gorm_tag.split(";"):
                if ":" in attr:
                    k, v = attr.split(":", 1)
                    gorm_attrs[k.strip()] = v.strip()
                else:
                    gorm_attrs[attr.strip()] = ""

        column = gorm_attrs.get("column", "")
        is_pk = "primaryKey" in gorm_attrs
        is_auto = "autoIncrement" in gorm_attrs

        not_null = False
        field_def = {
            "name": name,
            "go_type": go_type,
            "sql_type": GO_TYPE_SQL_MAP.get(go_type, "text"),
            "column": column or to_snake(name),
            "json_name": json_tag or to_snake(name),
            "is_pk": is_pk,
            "is_auto_increment": is_auto,
            "not_null": not_null,
            "default": gorm_attrs.get("default", ""),
        }
        fields.append(field_def)

    return fields


def to_snake(camel):
    s = re.sub(r"([a-z0-9])([A-Z])", r"\1_\2", camel)
    return s.lower()


def parse_seed_data():
    """Parse system_object_directory seed INSERT statements from db.sh."""
    db_sh = (
        SOURCE_DIR
        / "directory"
        / "ms_system_object_directory"
        / "dbshell"
        / "db.sh"
    )
    if not db_sh.exists():
        return []

    content = db_sh.read_text()
    blocks = content.split("insert into ${DB_TABLENAME}")
    entities = []
    for block in blocks[1:]:
        entity = _parse_insert_block(block)
        if entity:
            entities.append(entity)
    return entities


def _parse_insert_block(block):
    """Parse a single INSERT block into a dict mapping column names to values."""
    cols_start = block.find("(")
    cols_end = block.find(") values (")
    if cols_start < 0 or cols_end < 0:
        return None

    cols_text = block[cols_start + 1 : cols_end]
    columns = [c.strip() for c in cols_text.split(",")]

    val_start = cols_end + len(") values (")
    rest = block[val_start:]

    paren_depth = 1
    in_quote = False
    val_end = -1
    for i in range(len(rest)):
        ch = rest[i]
        if in_quote:
            if ch == "'":
                in_quote = False
            continue
        if ch == "'":
            in_quote = True
            continue
        if ch == "(":
            paren_depth += 1
        elif ch == ")":
            paren_depth -= 1
            if paren_depth == 0:
                val_end = i
                break

    if val_end < 0:
        return None

    values_text = rest[:val_end]

    paren_depth = 1
    in_quote = False
    current = []
    values = []
    for ch in values_text:
        if in_quote:
            current.append(ch)
            if ch == "'":
                in_quote = False
            continue
        if ch == "'":
            in_quote = True
            current.append(ch)
            continue
        if ch == "(":
            paren_depth += 1
            current.append(ch)
            continue
        if ch == ")":
            paren_depth -= 1
            current.append(ch)
            continue
        if ch == "," and paren_depth == 1:
            values.append(_parse_value("".join(current).strip()))
            current = []
            continue
        current.append(ch)
    if current:
        values.append(_parse_value("".join(current).strip()))

    result = {}
    for col, val in zip(columns, values):
        result[col] = val
    return result or None


def _parse_value(raw):
    """Parse a SQL literal value."""
    raw = raw.strip().rstrip(")")
    if raw.startswith("'") and raw.endswith("'"):
        return raw[1:-1]
    if raw == "TRUE":
        return True
    if raw == "FALSE":
        return False
    if raw == "":
        return ""
    try:
        return int(raw)
    except ValueError:
        pass
    try:
        return float(raw)
    except ValueError:
        pass
    return raw


def read_system_object_entities():
    """Read system_object_directory records from seed data, parsed into dicts."""
    import json
    seed = parse_seed_data()
    result = []
    for item in seed:
        if not isinstance(item, dict) or not item.get("name"):
            continue

        def json_or_str(val):
            if isinstance(val, str) and val.strip().startswith(("{", "[")):
                try:
                    return json.loads(val)
                except json.JSONDecodeError:
                    return val
            return val

        result.append({
            "name": item.get("name", ""),
            "description": item.get("description", ""),
            "status": item.get("status", 1),
            "prefix": item.get("prefix", 0),
            "prefix_number": item.get("prefix_number", ""),
            "level": item.get("level", 0),
            "is_group": item.get("is_group", False),
            "id_group": item.get("id_group", 0),
            "ref": item.get("ref", ""),
            "output_order": item.get("output_order", 0),
            "icon": item.get("icon", ""),
            "table_columns": json_or_str(item.get("table_columns", "")),
            "table_title": item.get("table_title", ""),
            "item_title": item.get("item_title", ""),
            "item_fields": json_or_str(item.get("item_fields", "")),
            "on_save_new": item.get("on_save_new", ""),
            "on_save_edit": item.get("on_save_edit", ""),
            "on_acceptance": item.get("on_acceptance", ""),
            "on_delete": item.get("on_delete", ""),
            "privileges": json_or_str(item.get("privileges", "")),
            "related_documents": item.get("related_documents", ""),
            "printing_template": item.get("printing_template", ""),
            "id_transaction": item.get("id_transaction", ""),
        })
    return result


def build_entity_index():
    """Build a index of: entity_ref -> {seed_info, microservice_info}."""
    entities_by_ref = {}

    for ent in read_system_object_entities():
        ref = ent.get("ref", "")
        if ref and ref != "#":
            entities_by_ref[ref] = {"seed": ent, "ms": None}

    for ms in read_all_microservices():
        ref = ms["ref"]
        if ref in entities_by_ref:
            entities_by_ref[ref]["ms"] = ms
        else:
            entities_by_ref[ref] = {"seed": seed_from_ref(ref), "ms": ms}

    return entities_by_ref


def seed_from_ref(ref):
    """Create a stub seed entry from microservice info when no seed data exists."""
    return {
        "name": ref.replace("_", " ").title(),
        "description": "",
        "ref": ref,
        "status": 1,
        "prefix": 0,
        "prefix_number": "",
        "level": 1,
        "is_group": False,
        "id_group": 1,
        "output_order": 0,
        "icon": "dictionary.png",
        "table_columns": None,
        "table_title": "",
        "item_title": "",
        "item_fields": None,
        "on_save_new": "",
        "on_save_edit": "",
        "on_acceptance": "",
        "on_delete": "",
        "privileges": None,
        "related_documents": "",
        "printing_template": "",
        "id_transaction": "",
    }
