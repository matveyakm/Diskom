/* Pilot: DOM inheritance via a customized built-in element.
 * <button is="dc-task-btn"> encapsulates title/icon/minimized rendering;
 * TaskbarManager only sets properties. Falls back to a plain <button>
 * where Custom Elements are unavailable. */
class TaskButton extends HTMLButtonElement {
  static get observedAttributes() {
    return ["minimized"];
  }

  constructor() {
    super();
    this.className = "taskbar-window-btn";
    this._icon = document.createElement("span");
    this._icon.className = "taskbar-window-icon";
    this._title = document.createElement("span");
    this._title.className = "taskbar-window-title";
    this.append(this._icon, this._title);
  }

  attributeChangedCallback(name, _old, value) {
    if (name === "minimized") {
      this.classList.toggle("minimized", value !== null);
    }
  }

  set label(text) {
    this._title.textContent = text;
    this.title = text;
  }

  set glyph(g) {
    this._icon.textContent = g || "📄";
  }

  set minimized(flag) {
    if (flag) {
      this.setAttribute("minimized", "");
    } else {
      this.removeAttribute("minimized");
    }
  }

  get minimized() {
    return this.hasAttribute("minimized");
  }
}

function createTaskButton() {
  if (
    typeof customElements !== "undefined" &&
    typeof document !== "undefined" &&
    typeof document.createElement === "function"
  ) {
    try {
      if (!customElements.get("dc-task-btn")) {
        customElements.define("dc-task-btn", TaskButton, { extends: "button" });
      }
      return document.createElement("button", { is: "dc-task-btn" });
    } catch (e) {
      console.warn("TaskButton: custom element unavailable, plain button:", e);
    }
  }
  const fallback = typeof document !== "undefined"
    ? document.createElement("button")
    : null;
  if (fallback) fallback.className = "taskbar-window-btn";
  return fallback;
}
