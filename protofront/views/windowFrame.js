/* Window chrome (single responsibility: frame, header/controls, drag,
 * resize, z-order, minimize/maximize/close). Listener references are stored
 * so removeEventListener actually detaches (fixes the .bind() leak). */
class WindowFrame {
  constructor({ title = "Window", icon = "📄" } = {}) {
    this.title = title;
    this.icon = icon;
    this.element = null;
    this.isDragging = false;
    this.dragStart = { x: 0, y: 0 };
    this.bounds = { width: 600, height: 400, left: 100, top: 100 };
    this.minimized = false;
    this.onDragMove = (e) => this.handleDrag(e);
    this.onDragEnd = () => this.endDrag();
    this.onClose = null;
    this.onMinimize = null;
    this.onRestore = null;
  }

  isMinimized() {
    return this.minimized;
  }

  static escapeHtml(text) {
    const div = document.createElement("div");
    div.textContent = text;
    return div.innerHTML;
  }

  static highestZIndex() {
    const windows = document.querySelectorAll(".desktop-window");
    let max = 1000;
    windows.forEach((win) => {
      const z = parseInt(win.style.zIndex);
      if (z > max) max = z;
    });
    return max;
  }

  mount(parent, { width, height, left, top } = {}) {
    Object.assign(this.bounds, { width, height, left, top });

    const root = document.createElement("div");
    root.className = "desktop-window";
    root.id =
      `window-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
    root.style.width = `${this.bounds.width}px`;
    root.style.height = `${this.bounds.height}px`;
    root.style.left = `${this.bounds.left}px`;
    root.style.top = `${this.bounds.top}px`;
    root.style.zIndex = WindowFrame.highestZIndex() + 1;

    const header = document.createElement("div");
    header.className = "window-header";
    header.innerHTML = `
      <div class="window-title">
        <span class="window-title-icon">${this.icon}</span>
        <span>${WindowFrame.escapeHtml(this.title)}</span>
      </div>
      <div class="window-controls">
        <button class="window-btn window-minimize" title="Свернуть">─</button>
        <button class="window-btn window-maximize" title="Развернуть">□</button>
        <button class="window-btn window-close" title="Закрыть">✕</button>
      </div>
    `;

    const content = document.createElement("div");
    content.className = "window-content";

    root.appendChild(header);
    root.appendChild(content);
    parent.appendChild(root);
    this.element = root;
    this.header = header;

    header
      .querySelector(".window-minimize")
      .addEventListener("click", () => this.minimize());
    header
      .querySelector(".window-maximize")
      .addEventListener("click", () => this.maximize());
    header
      .querySelector(".window-close")
      .addEventListener("click", () => this.close());
    header.addEventListener("mousedown", (e) => this.beginDrag(e));
    this.makeResizable();

    return content;
  }

  beginDrag(e) {
    if (e.target.closest(".window-controls")) return;
    this.isDragging = true;
    this.dragStart.x = e.clientX - this.element.offsetLeft;
    this.dragStart.y = e.clientY - this.element.offsetTop;
    this.element.style.cursor = "grabbing";
    this.bringToFront();
    document.addEventListener("mousemove", this.onDragMove);
    document.addEventListener("mouseup", this.onDragEnd);
  }

  handleDrag(e) {
    if (!this.isDragging) return;
    const newLeft = Math.max(
      0,
      Math.min(e.clientX - this.dragStart.x, window.innerWidth - this.element.offsetWidth),
    );
    const newTop = Math.max(
      0,
      Math.min(e.clientY - this.dragStart.y, window.innerHeight - this.element.offsetHeight),
    );
    this.element.style.left = `${newLeft}px`;
    this.element.style.top = `${newTop}px`;
  }

  endDrag() {
    this.isDragging = false;
    if (this.element) this.element.style.cursor = "";
    document.removeEventListener("mousemove", this.onDragMove);
    document.removeEventListener("mouseup", this.onDragEnd);
  }

  makeResizable() {
    const handle = document.createElement("div");
    handle.className = "window-resize-handle";
    this.element.appendChild(handle);

    let resizing = false;
    let startX, startY, startWidth, startHeight;
    const onMove = (e) => {
      if (!resizing) return;
      this.element.style.width = `${Math.max(300, startWidth + (e.clientX - startX))}px`;
      this.element.style.height = `${Math.max(200, startHeight + (e.clientY - startY))}px`;
    };
    const onUp = () => {
      resizing = false;
      document.removeEventListener("mousemove", onMove);
      document.removeEventListener("mouseup", onUp);
    };
    handle.addEventListener("mousedown", (e) => {
      resizing = true;
      startX = e.clientX;
      startY = e.clientY;
      startWidth = this.element.offsetWidth;
      startHeight = this.element.offsetHeight;
      document.addEventListener("mousemove", onMove);
      document.addEventListener("mouseup", onUp);
    });
  }

  minimize() {
    if (this.minimized || !this.element) return;
    this.minimized = true;
    this.element.classList.add("minimized");
    setTimeout(() => {
      if (this.element) this.element.style.display = "none";
    }, 300);
    if (this.onMinimize) this.onMinimize();
  }

  restore() {
    if (!this.minimized || !this.element) return;
    this.minimized = false;
    this.element.classList.remove("minimized");
    this.element.style.display = "";
    this.bringToFront();
    if (this.onRestore) this.onRestore();
  }

  maximize() {
    if (this.element.classList.contains("maximized")) {
      this.element.classList.remove("maximized");
      this.element.style.width = `${this.bounds.width}px`;
      this.element.style.height = `${this.bounds.height}px`;
      this.element.style.left = `${this.bounds.left}px`;
      this.element.style.top = `${this.bounds.top}px`;
    } else {
      this.element.classList.add("maximized");
      this.element.style.width = "100%";
      this.element.style.height = `calc(100vh - 48px)`;
      this.element.style.left = "0";
      this.element.style.top = "0";
    }
  }

  bringToFront() {
    this.element.style.zIndex = WindowFrame.highestZIndex() + 1;
  }

  close() {
    if (!this.element) return;
    const el = this.element;
    this.element = null;
    el.classList.add("closing");
    setTimeout(() => {
      el.remove();
      if (this.onClose) this.onClose();
    }, 300);
  }
}
