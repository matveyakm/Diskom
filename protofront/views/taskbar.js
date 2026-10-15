/* Taskbar manager (single responsibility: taskbar buttons for open windows).
 * One entry per window: click toggles minimize/restore. The manager knows
 * nothing about window internals — entries expose a small interface:
 * { id, title, icon, isMinimized(), minimize(), restore() }.
 * Lifecycle (untrack on close) stays with the window composer. */
class TaskbarManager {
  constructor(taskbarEl) {
    this.taskbarEl = taskbarEl;
    this.entries = new Map();
    this.container = document.createElement("div");
    this.container.className = "taskbar-windows";
    // Insert after the start button, before the clock (#time has margin-left:auto).
    const clock = taskbarEl.querySelector("#time");
    taskbarEl.insertBefore(this.container, clock);
  }

  static global() {
    if (!TaskbarManager.instance) {
      const taskbar = document.getElementById("taskbar");
      if (!taskbar) return null;
      TaskbarManager.instance = new TaskbarManager(taskbar);
    }
    return TaskbarManager.instance;
  }

  track(entry) {
    if (this.entries.has(entry.id)) return;
    const btn = createTaskButton();
    btn.label = entry.title;
    btn.glyph = entry.icon;
    btn.addEventListener("click", () => {
      if (entry.isMinimized()) {
        entry.restore();
      } else {
        entry.minimize();
      }
    });
    this.container.appendChild(btn);
    const record = { entry, btn };
    this.entries.set(entry.id, record);
    this.refresh(record);
    return record;
  }

  untrack(id) {
    const record = this.entries.get(id);
    if (!record) return;
    record.btn.remove();
    this.entries.delete(id);
  }

  refresh(record) {
    // Custom-element path reflects via attribute; plain-button fallback
    // toggles the class directly.
    const isCustom =
      typeof TaskButton !== "undefined" && record.btn instanceof TaskButton;
    if (isCustom) {
      record.btn.minimized = record.entry.isMinimized();
    } else {
      record.btn.classList.toggle("minimized", record.entry.isMinimized());
    }
  }

  refreshById(id) {
    const record = this.entries.get(id);
    if (record) this.refresh(record);
  }

  refreshAll() {
    this.entries.forEach((record) => this.refresh(record));
  }
}

TaskbarManager.instance = null;
