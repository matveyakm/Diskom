#!/usr/bin/env python3
"""Generate ER-diagram sources for the system_object metadata DB.

Parses the authoritative DDL from dbshell/db.sh.base, adds the logical
relationships that exist in the seeds but are not enforced by a constraint,
and writes:

  system_object_er.mmd   Mermaid ER source (key columns: PK/FK/name)
  system_object_er.html  standalone viewer (Mermaid from CDN)
  system_object_er.dot   detailed Graphviz source (all columns)

Rasterize the Mermaid view (the committed PNG was produced this way):

    google-chrome --headless=new --no-sandbox --hide-scrollbars \\
      --virtual-time-budget=20000 --window-size=4200,2600 \\
      --screenshot=system_object_er.png "file://$PWD/system_object_er.html"
    convert system_object_er.png -trim +repage system_object_er.png

Alternative (Graphviz):

    dot -Tsvg system_object_er.dot -o system_object_er.svg
    dot -Tpng -Gdpi=160 system_object_er.dot -o system_object_er.png
"""
from __future__ import annotations

import html
import pathlib
import re

HERE = pathlib.Path(__file__).resolve().parent
ROOT = HERE.parents[1]
DDL = ROOT / "source/directory/ms_system_object_directory/dbshell/db.sh.base"

SKIP_PREFIXES = ("UNIQUE", "PRIMARY", "CONSTRAINT", "FOREIGN", "CHECK")

GROUPS = {
    "catalog": {
        "object_type", "object_status", "object_icon", "object_link", "system_object",
    },
    "forms": {
        "form_type", "object_fields", "object_forms", "tab_forms", "fields_tab",
        "field_editor_type",
    },
    "workflow": {
        "events_type", "method_type", "workflow_process", "filter_set",
        "filter_set_item", "filter_type", "map_set", "map_set_item", "value_types",
        "fk_set", "fk_set_item", "business_process", "workflow_in_process",
    },
    "relations": {"related_document", "privilege", "privilege_types"},
}
GROUP_COLOR = {
    "catalog": "#1f4e79",
    "forms": "#2e6b3a",
    "workflow": "#8a4b08",
    "relations": "#5b2d82",
}
GROUP_OF = {t: g for g, ts in GROUPS.items() for t in ts}

# FKs that exist logically (columns are populated by the seeds) but have no
# physical FK constraint in the DDL.
LOGICAL_FKS = {
    ("object_fields", "system_object", "system_object", "id"),
    ("object_fields", "object_type", "object_type", "id"),
    ("object_forms", "system_object", "system_object", "id"),
    ("object_forms", "form_type", "form_type", "id"),
    ("tab_forms", "object_form", "object_forms", "id"),
    ("fields_tab", "tab_form", "tab_forms", "id"),
    ("fields_tab", "object_field", "object_fields", "id"),
    ("map_set_item", "server_field", "object_fields", "id"),
    ("map_set_item", "application_field", "object_fields", "id"),
    ("workflow_process", "object_form", "object_forms", "id"),
    ("workflow_process", "events_type", "events_type", "id"),
    ("workflow_process", "method_type", "method_type", "id"),
    ("workflow_in_process", "workflow_process", "workflow_process", "id"),
    ("workflow_in_process", "business_process", "business_process", "id"),
}

# Physical FKs that should not force the child down a rank (long cross links).
NO_RANK = {
    ("fields_tab", "editor_ref", "system_object", "id"),
    ("map_set_item", "server_field", "object_fields", "id"),
    ("map_set_item", "application_field", "object_fields", "id"),
}

COLUMN_LABEL = {"default": "default"}


def parse_ddl(text: str):
    tables: dict[str, list[dict]] = {}
    for m in re.finditer(r"CREATE TABLE public\.(\w+) \((.*?)\n\s*\);", text, re.S):
        name, body = m.group(1), m.group(2)
        cols: list[dict] = []
        for raw in body.splitlines():
            line = raw.strip().rstrip(",")
            if not line or line.startswith(SKIP_PREFIXES):
                continue
            cm = re.match(r'"?([a-z_]+)"?\s+(.*)$', line)
            if not cm:
                continue
            col, rest = cm.group(1), cm.group(2)
            typ = rest.split()[0].lower()
            ref = re.search(r"REFERENCES public\.(\w+)\((\w+)\)", rest)
            cols.append({
                "name": col,
                "type": typ,
                "pk": "PRIMARY KEY" in rest,
                "ref": (ref.group(1), ref.group(2)) if ref else None,
            })
        tables[name] = cols

    physical = set()
    for m in re.finditer(
        r"ALTER TABLE public\.(\w+) ADD CONSTRAINT \w+\s+"
        r"FOREIGN KEY \((\w+)\)\s+REFERENCES public\.(\w+)\((\w+)\)",
        text,
    ):
        physical.add((m.group(1), m.group(2), m.group(3), m.group(4)))
    for t, cols in tables.items():
        for c in cols:
            if c["ref"]:
                physical.add((t, c["name"], c["ref"][0], c["ref"][1]))
    return tables, physical


def esc(s: str) -> str:
    return html.escape(s, quote=True)


def node_dot(name: str, cols: list[dict], fk_cols: set[str]) -> str:
    color = GROUP_COLOR[GROUP_OF.get(name, "catalog")]
    rows = [
        f'<TR><TD COLSPAN="2" BGCOLOR="{color}" ALIGN="LEFT" CELLPADDING="4">'
        f'<FONT COLOR="white" POINT-SIZE="12"><B>{esc(name)}</B></FONT></TD></TR>'
    ]
    for c in cols:
        marks = []
        if c["pk"]:
            marks.append('<FONT COLOR="#b8860b" POINT-SIZE="8"><B>PK</B></FONT>')
        if c["name"] in fk_cols:
            marks.append('<FONT COLOR="#1f4e79" POINT-SIZE="8"><B>FK</B></FONT>')
        marker = " " + " ".join(marks) if marks else ""
        rows.append(
            f'<TR><TD PORT="{esc(c["name"])}" ALIGN="LEFT" CELLPADDING="3">'
            f'<FONT POINT-SIZE="10">{esc(c["name"])}</FONT></TD>'
            f'<TD ALIGN="LEFT" CELLPADDING="3">'
            f'<FONT COLOR="#666666" POINT-SIZE="9">{esc(c["type"])}</FONT>{marker}</TD></TR>'
        )
    label = (
        '<<TABLE BORDER="0" CELLBORDER="1" CELLSPACING="0" CELLPADDING="0" '
        'COLOR="#b8b8b8" BGCOLOR="white">' + "".join(rows) + "</TABLE>>"
    )
    return (
        f'  "{name}" [shape=plain label={label}];\n'
    )


def edge_dot(child, ccol, parent, pcol, physical: bool) -> str:
    attrs = [
        "dir=both",
        "arrowtail=tee",
        "arrowhead=crow",
        "penwidth=1.1",
    ]
    if physical:
        attrs += ['color="#333333"']
    else:
        attrs += ['color="#8a8a8a"', "style=dashed"]
    if (child, ccol, parent, pcol) in NO_RANK:
        attrs += ["constraint=false"]
    if child == parent:
        attrs += ["constraint=false"]
    return f'  "{parent}":"{pcol}" -> "{child}":"{ccol}" [{", ".join(attrs)}];\n'


def emit_mermaid(tables, physical, path: pathlib.Path) -> None:
    """Mermaid ER source: key columns only (PK/FK/name) for a readable map."""
    fkcols = {(ch, cc) for ch, cc, _, _ in physical | LOGICAL_FKS}
    lines = ["erDiagram"]
    for name in sorted(tables, key=lambda n: (GROUP_OF.get(n, "zz"), n)):
        lines.append(f"  {name} {{")
        for c in tables[name]:
            is_pk = c["pk"]
            is_fk = (name, c["name"]) in fkcols
            if not (is_pk or is_fk or c["name"] == "name"):
                continue
            keys = " ".join(k for k, ok in (("PK", is_pk), ("FK", is_fk)) if ok)
            lines.append(f"    {c['type']} {c['name']} {keys}".rstrip())
        lines.append("  }")

    seen: set[tuple[str, str, str]] = set()
    for child, ccol, parent, _ in sorted(physical):
        if (parent, child, ccol) in seen:
            continue
        seen.add((parent, child, ccol))
        lines.append(f'  {parent} ||--o{{ {child} : "{ccol}"')
    for child, ccol, parent, _ in sorted(set(LOGICAL_FKS) - physical):
        if (parent, child, ccol) in seen:
            continue
        seen.add((parent, child, ccol))
        lines.append(f'  {parent} ||..o{{ {child} : "{ccol}"')
    path.write_text("\n".join(lines) + "\n")


def emit_html(mmd_path: pathlib.Path, html_path: pathlib.Path) -> None:
    """Standalone viewer (Mermaid from CDN) used to rasterize the PNG."""
    mmd = mmd_path.read_text()
    html_path.write_text(
        """<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<title>system_object — metadata DB (ER)</title>
<style>
  body { margin: 0; background: #fff; font-family: "DejaVu Sans", sans-serif; color: #1c2b3a; }
  header { padding: 20px 26px 8px; }
  h1 { margin: 0 0 6px; font-size: 48px; font-weight: 600; }
  header p { margin: 0; font-size: 18px; color: #4a5a6a; }
  code { font-family: "DejaVu Sans Mono", monospace; }
  pre.mermaid { margin: 0; }
</style>
<script type="module">
import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs';
mermaid.initialize({
  startOnLoad: true,
  theme: 'base',
  themeVariables: {
    fontFamily: 'DejaVu Sans, sans-serif',
    fontSize: '14px',
    primaryColor: '#eef3f8',
    primaryBorderColor: '#9fb3c8',
    primaryTextColor: '#1c2b3a',
    lineColor: '#4a5a6a',
    tertiaryColor: '#ffffff'
  },
  er: { diagramPadding: 24, minEntityWidth: 120, minEntityHeight: 40,
        entityPadding: 10, useMaxWidth: false }
});
</script></head>
<body>
<header>
  <h1>system_object — metadata database</h1>
  <p><b>Solid</b> — physical FK &middot; <b>Dashed</b> — logical FK (seeded, not enforced)
     &middot; crow's foot = many &middot;
     <code>system_object_directory</code> is a materialized read model over these tables.</p>
</header>
<pre class="mermaid">
""" + mmd + """</pre>
</body></html>
"""
    )


def main() -> None:
    tables, physical = parse_ddl(DDL.read_text())
    fk_cols: dict[str, set[str]] = {t: set() for t in tables}
    for t, c, _, _ in physical | LOGICAL_FKS:
        fk_cols.setdefault(t, set()).add(c)

    dot = [
        "digraph system_object {",
        '  graph [rankdir=LR, splines=true, overlap=false, nodesep=0.28, '
        'ranksep=1.0, pad=0.35, bgcolor="white", fontname="DejaVu Sans"];',
        '  node [shape=plain, fontname="DejaVu Sans", fontsize=10];',
        '  edge [fontname="DejaVu Sans", fontsize=8];',
        '  label=<<TABLE BORDER="0" CELLBORDER="0" CELLSPACING="4" CELLPADDING="2">'
        '<TR><TD ALIGN="LEFT"><FONT POINT-SIZE="22"><B>system_object — metadata DB</B></FONT></TD></TR>'
        '<TR><TD ALIGN="LEFT"><FONT POINT-SIZE="10" COLOR="#444444">'
        'Solid = physical FK constraint · Dashed = logical FK (seeded, not enforced) · '
        'Crow\'s foot = many · Bar = one</FONT></TD></TR>'
        '<TR><TD ALIGN="LEFT"><FONT POINT-SIZE="9" COLOR="#666666">'
        'catalog</FONT> <FONT COLOR="#1f4e79">■</FONT>  '
        '<FONT POINT-SIZE="9" COLOR="#666666">forms/fields</FONT> <FONT COLOR="#2e6b3a">■</FONT>  '
        '<FONT POINT-SIZE="9" COLOR="#666666">workflow</FONT> <FONT COLOR="#8a4b08">■</FONT>  '
        '<FONT POINT-SIZE="9" COLOR="#666666">relations</FONT> <FONT COLOR="#5b2d82">■</FONT>'
        '</FONT></TD></TR></TABLE>>;',
        '  labelloc="t";',
        '  labeljust="l";',
    ]

    for name in sorted(tables, key=lambda n: (GROUP_OF.get(n, "zz"), n)):
        dot.append(node_dot(name, tables[name], fk_cols[name]))

    # Read model (materialized view) — no edges into the layout.
    dot.append(
        '  "system_object_directory" [shape=plain label=<<TABLE BORDER="1" '
        'CELLBORDER="1" CELLSPACING="0" CELLPADDING="3" COLOR="#999999" '
        'BGCOLOR="#f4f4f4" STYLE="dashed"><TR><TD BGCOLOR="#555555" ALIGN="LEFT">'
        '<FONT COLOR="white" POINT-SIZE="12"><B>system_object_directory</B></FONT></TD></TR>'
        '<TR><TD ALIGN="LEFT"><FONT POINT-SIZE="9">MATERIALIZED VIEW</FONT></TD></TR>'
        '<TR><TD ALIGN="LEFT"><FONT POINT-SIZE="9">read model over the tables above</FONT></TD></TR>'
        '</TABLE>>];\n'
    )

    for e in sorted(physical):
        dot.append(edge_dot(*e, True))
    for e in sorted(LOGICAL_FKS - physical):
        dot.append(edge_dot(*e, False))

    dot.append(
        '  "system_object" -> "system_object_directory" '
        '[style=dotted, color="#555555", arrowhead=none, constraint=false, '
        'label="  materializes  ", fontsize=9];\n'
    )
    dot.append("}\n")
    out = HERE / "system_object_er.dot"
    out.write_text("".join(dot))
    mmd = HERE / "system_object_er.mmd"
    emit_mermaid(tables, physical, mmd)
    emit_html(mmd, HERE / "system_object_er.html")
    print(out)
    print(mmd)
    print(HERE / "system_object_er.html")


if __name__ == "__main__":
    main()
