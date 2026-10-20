#!/usr/bin/env python3
"""Rewrite 02_system_object_relational.sql seeds: scalar subqueries -> CTE joins.

Usage: python3 Dev_tools/rewrite_seed_joins.py [--dry] <file.sql> [...]
"""
import re
import sys
import pathlib

FIRST = re.compile(r'^(?:WITH )?([a-z_0-9]+) AS \(', re.M)

EV = """ev AS (
  SELECT
    (SELECT id FROM events_type WHERE name='onSaveNew')    AS save_new,
    (SELECT id FROM events_type WHERE name='onSaveEdit')   AS save_edit,
    (SELECT id FROM events_type WHERE name='onAcceptance') AS acceptance,
    (SELECT id FROM events_type WHERE name='onDelete')     AS delete_ev
)"""

MT = """mt AS (
  SELECT
    (SELECT id FROM method_type WHERE name='post')   AS post,
    (SELECT id FROM method_type WHERE name='put')    AS put,
    (SELECT id FROM method_type WHERE name='get')    AS get,
    (SELECT id FROM method_type WHERE name='delete') AS delete_m
)"""

FK_ROW = re.compile(
    r"SELECT \(SELECT id FROM (tab\d+)\),\(SELECT id FROM fld WHERE name='([^']+)'\),"
    r"'([^']*)',(TRUE|FALSE),(TRUE|FALSE),"
    r"(NULL::bigint|\(SELECT id FROM field_editor_type WHERE name='([^']+)'\)),"
    r"(NULL::bigint|\(SELECT so2\.id FROM system_object so2 JOIN object_link ol ON ol\.id=so2\.object_link WHERE ol\.path='([^']+)'\)),"
    r"'([^']*)'(?:,(TRUE|FALSE))?"
)

WP = re.compile(
    r"INSERT INTO workflow_process\(name, object_form, events_type, method_type, step, end_point, filter_set, hide_in_preview, conditions, set_to_server, fk_set, fields\)\s*"
    r"SELECT '([^']*)',\s*\(SELECT id FROM frm_item\),\s*\(SELECT id FROM events_type WHERE name='([^']+)'\),\s*\(SELECT id FROM method_type WHERE name='([^']+)'\),\s*"
    r"(\d+),\s*'([^']*)',\s*(NULL|\(SELECT id FROM (\w+)\)),\s*(TRUE|FALSE),\s*'([^']*)',\s*(NULL|\(SELECT id FROM (\w+)\)),\s*(NULL|\(SELECT id FROM (\w+)\)),\s*'([^']*)'\s*RETURNING id\)",
    re.S,
)

EV_KEYS = {'onSaveNew': 'save_new', 'onSaveEdit': 'save_edit',
           'onAcceptance': 'acceptance', 'onDelete': 'delete_ev'}
MT_KEYS = {'post': 'post', 'put': 'put', 'get': 'get', 'delete': 'delete_m'}


def split_blocks(text):
    ms = list(FIRST.finditer(text))
    if not ms:
        return None
    header = text[:ms[0].start()]
    raw = []
    for i, m in enumerate(ms):
        end = ms[i + 1].start() if i + 1 < len(ms) else len(text)
        raw.append((m.group(1), text[m.start():end]))
    name, b = raw[-1]
    ti = b.find('\nSELECT ')
    if ti < 0:
        raise SystemExit('no tail in ' + name)
    tail = b[ti + 1:]
    raw[-1] = (name, b[:ti + 1])
    return header, raw, tail


def clean(b):
    lines = [ln for ln in b.strip().split('\n') if not ln.lstrip().startswith('--')]
    b = '\n'.join(lines).strip()
    if b.endswith(','):
        b = b[:-1].rstrip()
    return b


def gen_fld(block):
    names = re.findall(
        r"SELECT \(SELECT id FROM so\),'([^']+)',\(SELECT id FROM object_type WHERE name='Text'\)",
        block)
    if not names:
        return clean(block)
    rows = ',\n    '.join(f"({i + 1},'{n}')" for i, n in enumerate(names))
    return (
        "fld AS (\n"
        "  INSERT INTO object_fields(system_object,name,object_type)\n"
        "  SELECT so.id, v.name, ot.id\n"
        "  FROM so, (VALUES\n"
        f"    {rows}) AS v(ord,name), object_type ot\n"
        "  WHERE ot.name='Text'\n"
        "  ORDER BY v.ord\n"
        "  RETURNING id,name)"
    )


def gen_form(block, name):
    m = re.search(
        r"INSERT INTO object_forms\(system_object,name,form_type\) SELECT \(SELECT id FROM so\),'([^']*)',\(SELECT id FROM form_type WHERE name='([^']+)'\) RETURNING id\)",
        block)
    if not m:
        raise SystemExit('form parse fail ' + name)
    title, ftype = m.group(1), m.group(2)
    return (
        f"{name} AS (\n"
        f"  INSERT INTO object_forms(system_object,name,form_type) SELECT so.id,'{title}',fty.id FROM so, form_type fty WHERE fty.name='{ftype}' RETURNING id)"
    )


def gen_tab(block, name):
    m = re.search(
        r"INSERT INTO tab_forms\(name,title,tag,object_form,output_order,visible\) SELECT '([^']*)','([^']*)',(NULL|'[^']*'),\(SELECT id FROM (frm_\w+)\),(\d+),(TRUE|FALSE) RETURNING id\)",
        block)
    if not m:
        raise SystemExit('tab parse fail ' + name)
    tname, title, tag, form, order, vis = m.groups()
    return (
        f"{name} AS (\n"
        f"  INSERT INTO tab_forms(name,title,tag,object_form,output_order,visible) SELECT '{tname}','{title}',{tag},{form}.id,{order},{vis} FROM {form} RETURNING id)"
    )


def gen_tabs(tabnames):
    parts = [f"SELECT '{n}' AS name, id FROM {n}" for n in tabnames]
    return "tabs AS (" + " UNION ALL ".join(parts) + ")"


def gen_ft(block, has_ro):
    rows = FK_ROW.findall(block)
    if not rows:
        raise SystemExit('no ft rows')
    vals = []
    for i, r in enumerate(rows):
        tab, fname, label, vis, req, ed, edname, er, erpath, dflt, ro = r
        edv = f"'{edname}'" if ed != 'NULL::bigint' else 'NULL::text'
        erv = f"'{erpath}'" if er != 'NULL::bigint' else 'NULL::text'
        base = f"({i + 1},'{tab}','{fname}','{label}',{vis},{req},{edv},{erv},'{dflt}'"
        if has_ro:
            base += f",{ro or 'FALSE'}"
        base += ")"
        vals.append(base)
    cols = 'tab_form,object_field,label,visible,required,editor_type,editor_ref,"default"'
    sel = 't.id, f.id, v.label, v.visible, v.required, fet.id, ros.id, v.dflt'
    valcols = 'ord,tab,fname,label,visible,required,editor,eref,dflt'
    if has_ro:
        cols += ',readonly'
        sel += ', v.readonly'
        valcols += ',readonly'
    return (
        "ft AS (\n"
        f"  INSERT INTO fields_tab({cols})\n"
        f"  SELECT {sel}\n"
        "  FROM (VALUES\n"
        f"    {',\n    '.join(vals)}) AS v({valcols})\n"
        "  JOIN tabs t ON t.name = v.tab\n"
        "  JOIN fld f ON f.name = v.fname\n"
        "  LEFT JOIN field_editor_type fet ON fet.name = v.editor\n"
        "  LEFT JOIN object_link olr ON olr.path = v.eref\n"
        "  LEFT JOIN system_object ros ON ros.object_link = olr.id\n"
        "  ORDER BY v.ord\n"
        "  RETURNING id)"
    )


def gen_flt_i(block, name):
    m = re.search(
        r"INSERT INTO filter_set_item\(filter_set, filter_type, output_order\)\s*SELECT \(SELECT id FROM (\w+)\), \(SELECT id FROM filter_type WHERE name='([^']+)'\), (\d+) RETURNING filter_set\)",
        block, re.S)
    if not m:
        raise SystemExit('flt parse fail ' + name)
    ref, ftype, order = m.groups()
    return (
        f"{name} AS (\n"
        "  INSERT INTO filter_set_item(filter_set, filter_type, output_order)\n"
        f"  SELECT {ref}.id, fty.id, {order} FROM {ref}, filter_type fty WHERE fty.name='{ftype}' RETURNING filter_set)"
    )


def reformat_rows(rows):
    lines = [ln.strip().rstrip(',') for ln in rows.strip().splitlines() if ln.strip()]
    return '    ' + ',\n    '.join(lines)


def gen_map_i(block, name):
    m = re.search(
        r"INSERT INTO map_set_item\(map_set, target_field, send_value, output_order\)\s*SELECT \(SELECT id FROM (\w+)\), v\.t, v\.s, v\.o FROM \(VALUES\n(.*?)\) AS v\(t,s,o\)\s*RETURNING map_set\)",
        block, re.S)
    if not m:
        raise SystemExit('map parse fail ' + name)
    ref, rows = m.groups()
    return (
        f"{name} AS (\n"
        "  INSERT INTO map_set_item(map_set, target_field, send_value, output_order)\n"
        f"  SELECT {ref}.id, v.t, v.s, v.o FROM {ref}, (VALUES\n{reformat_rows(rows)}) AS v(t,s,o)\n"
        "  ORDER BY v.o\n"
        "  RETURNING map_set)"
    )


def gen_fk_i(block, name):
    m = re.search(
        r"INSERT INTO fk_set_item\(fk_set, context_field, response_field, output_order\)\s*SELECT \(SELECT id FROM (\w+)\), v\.c, v\.r, v\.o FROM \(VALUES\n(.*?)\) AS v\(c,r,o\)\s*RETURNING fk_set\)",
        block, re.S)
    if m:
        ref, rows = m.groups()
        return (
            f"{name} AS (\n"
            "  INSERT INTO fk_set_item(fk_set, context_field, response_field, output_order)\n"
            f"  SELECT {ref}.id, v.c, v.r, v.o FROM {ref}, (VALUES\n{reformat_rows(rows)}) AS v(c,r,o)\n"
            "  ORDER BY v.o\n"
            "  RETURNING fk_set)"
        )
    m = re.search(
        r"INSERT INTO fk_set_item\(fk_set, context_field, response_field, output_order\)\s*SELECT \(SELECT id FROM (\w+)\), '([^']*)', '([^']*)', (\d+) RETURNING fk_set\)",
        block, re.S)
    if not m:
        raise SystemExit('fk parse fail ' + name)
    ref, c, r, o = m.groups()
    return (
        f"{name} AS (\n"
        "  INSERT INTO fk_set_item(fk_set, context_field, response_field, output_order)\n"
        f"  SELECT {ref}.id, '{c}', '{r}', {o} FROM {ref} RETURNING fk_set)"
    )


def split_top(s):
    parts = []
    depth = 0
    inq = False
    cur = []
    i = 0
    while i < len(s):
        c = s[i]
        if inq:
            cur.append(c)
            if c == "'":
                if i + 1 < len(s) and s[i + 1] == "'":
                    cur.append(s[i + 1]); i += 2; continue
                inq = False
            i += 1; continue
        if c == "'":
            inq = True; cur.append(c); i += 1; continue
        if c == '(':
            depth += 1
        elif c == ')':
            depth -= 1
        if c == ',' and depth == 0:
            parts.append(''.join(cur).strip()); cur = []; i += 1; continue
        cur.append(c); i += 1
    parts.append(''.join(cur).strip())
    return parts


def gen_wp(block, name):
    m = re.search(r"INSERT INTO workflow_process\(([^)]*)\)\s*SELECT (.*?)\s*RETURNING id\)", block, re.S)
    if not m:
        raise SystemExit('wp parse fail ' + name)
    cols = [c.strip() for c in m.group(1).split(',')]
    exprs = split_top(m.group(2))
    if len(cols) != len(exprs):
        raise SystemExit(f'wp col/expr mismatch {name}: {len(cols)} vs {len(exprs)}')
    froms = []
    out = []
    for col, expr in zip(cols, exprs):
        if col == 'object_form':
            if expr == 'NULL':
                out.append('NULL')
            else:
                out.append('frm_item.id'); froms.append('frm_item')
        elif col == 'events_type':
            mm = re.search(r"\(SELECT (\w+) FROM ev\)", expr)
            if mm:
                out.append('ev.' + mm.group(1))
            elif expr == 'NULL':
                out.append('NULL')
            else:
                mm = re.search(r"events_type WHERE name='([^']+)'", expr)
                out.append('ev.' + EV_KEYS[mm.group(1)])
            if expr != 'NULL':
                froms.append('ev')
        elif col == 'method_type':
            mm = re.search(r"\(SELECT (\w+) FROM mt\)", expr)
            if mm:
                out.append('mt.' + mm.group(1))
            elif expr == 'NULL':
                out.append('NULL')
            else:
                mm = re.search(r"method_type WHERE name='([^']+)'", expr)
                out.append('mt.' + MT_KEYS[mm.group(1)])
            if expr != 'NULL':
                froms.append('mt')
        elif col in ('filter_set', 'set_to_server', 'fk_set'):
            mm = re.match(r"\(SELECT id FROM (\w+)\)$", expr)
            if mm:
                out.append(mm.group(1) + '.id'); froms.append(mm.group(1))
            else:
                out.append(expr)
        else:
            out.append(expr)
    seen = set(); fl = []
    for f in froms:
        if f not in seen:
            seen.add(f); fl.append(f)
    return (
        f"{name} AS (\n"
        f"  INSERT INTO workflow_process({m.group(1)})\n"
        f"  SELECT {', '.join(out)} FROM {', '.join(fl)} RETURNING id)"
    )


def gen_priv(block):
    m = re.search(r"pt\.name IN \((.*?)\)", block, re.S)
    if not m:
        raise SystemExit('priv parse fail')
    names = m.group(1)
    return (
        "priv AS (\n"
        "  INSERT INTO privilege(system_object, privilege_type, output_order)\n"
        "  SELECT so.id, pt.id, pt.id FROM so, privilege_types pt\n"
        f"  WHERE pt.name IN ({names}) RETURNING id)"
    )


def transform(text):
    parsed = split_blocks(text)
    if parsed is None:
        return None
    header, blocks, tail = parsed
    has_ro = bool(re.search(r'INSERT INTO fields_tab\([^)]*readonly', text))
    names = [n for n, _ in blocks]
    has_wp = any(n.startswith('wp_') for n in names)
    add_ev = has_wp and 'ev' not in names
    add_mt = has_wp and 'mt' not in names
    out = []
    seen_tabs = []
    tabs_done = False
    for name, block in blocks:
        if not tabs_done and seen_tabs and not re.fullmatch(r'tab\d+', name):
            out.append(gen_tabs(seen_tabs))
            tabs_done = True
        if name == 'so':
            out.append(clean(block))
        elif name == 'fld':
            out.append(gen_fld(block))
        elif name in ('frm_list', 'frm_item'):
            out.append(gen_form(block, name))
        elif re.fullmatch(r'tab\d+', name):
            seen_tabs.append(name)
            out.append(gen_tab(block, name))
        elif name == 'ft':
            out.append(gen_ft(block, has_ro))
            if add_ev:
                out.append(EV)
            if add_mt:
                out.append(MT)
        elif 'INSERT INTO filter_set_item' in block:
            out.append(gen_flt_i(block, name))
        elif 'INSERT INTO map_set_item' in block:
            out.append(gen_map_i(block, name))
        elif 'INSERT INTO fk_set_item' in block:
            out.append(gen_fk_i(block, name))
        elif name.startswith('wp_'):
            out.append(gen_wp(block, name))
        elif name == 'priv':
            out.append(gen_priv(block))
        else:
            out.append(clean(block))
    tail = tail.replace('SELECT (SELECT id FROM so) AS system_object_id;',
                        'SELECT so.id AS system_object_id FROM so;')
    return header + ',\n'.join(out) + '\n' + tail


def main():
    args = sys.argv[1:]
    dry = '--dry' in args
    args = [a for a in args if a != '--dry']
    for a in args:
        p = pathlib.Path(a)
        old = p.read_text()
        new = transform(old)
        if new is None:
            print('SKIP (no WITH):', p)
            continue
        if dry:
            sys.stdout.write(new)
        else:
            p.write_text(new)
            print('rewrote', p)


if __name__ == '__main__':
    main()
