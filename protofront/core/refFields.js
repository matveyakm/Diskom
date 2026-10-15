/* Reference-field helpers (single responsibility: new editor/editor_ref
 * shape with legacy type:"select"+end_point fallback). */
function refOf(col) {
  if (!col) return undefined;
  return col.editor_ref ?? col.end_point;
}

function isRefField(col) {
  if (!col) return false;
  if (col.editor === "dropdown" || col.editor === "form") return true;
  return col.type === "select" && !!refOf(col);
}
