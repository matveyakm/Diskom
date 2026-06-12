class MenuManager {
  constructor(menuElement, contentElement) {
    this.menu = menuElement;
    this.content = contentElement;
    this.isLoaded = false;
  }

  open() {
    this.menu.classList.add("active");
  }

  close() {
    this.menu.classList.remove("active");
  }

  toggle() {
    this.menu.classList.toggle("active");
  }

  isActive() {
    return this.menu.classList.contains("active");
  }

  async loadAndRender(fetchDataFn) {
    if (this.isLoaded) return;

    try {
      const data = await fetchDataFn();
      this.render(data);
      this.isLoaded = true;
    } catch (err) {
      this.showError();
    }
  }

  render(items) {
    this.content.innerHTML = "";
    const ul = document.createElement("ul");
    ul.className = "menu-list";
    this.renderLevel(items, ul);
    this.content.appendChild(ul);
  }

  renderLevel(items, parentUl, parentId = 0, parentLevel = -1) {
    const nextLevel = parentLevel + 1;
    const children = items.filter(
      (item) =>
        item.level === nextLevel &&
        (item.id_group === parentId || item.id_group === 0),
    );
    children.forEach((child) => {
      const li = this.createMenuItem(child, items, child.id, nextLevel);
      parentUl.appendChild(li);
    });
  }

  createMenuItem(item, allItems, parentId, currentLevel) {
    const li = document.createElement("li");
    li.className = "menu-item";
    li.id = item.id;
    li.dataset.level = item.level;

    const contentDiv = document.createElement("div");
    contentDiv.className = "menu-item-content";

    // FIX: consistent lowercase "icon" for both check and value
    if (item.icon) {
      const img = document.createElement("img");
      img.src = `png/${item.icon}`;
      img.className = "menu-item-icon";
      img.alt = item.name;
      contentDiv.appendChild(img);
    }

    const span = document.createElement("span");
    span.textContent = item.name;
    contentDiv.appendChild(span);

    li.appendChild(contentDiv);

    const hasChildren = allItems.some(
      (i) => i.id_group === item.id && i.level === currentLevel + 1,
    );

    if (hasChildren) {
      li.classList.add("has-submenu");

      const subUl = document.createElement("ul");
      subUl.className = "menu-list";

      contentDiv.addEventListener("mouseenter", (e) => {
        e.stopPropagation();
        document.querySelectorAll(".menu-item.open").forEach((openItem) => {
          if (openItem !== li) openItem.classList.remove("open");
        });
        li.classList.add("open"); // FIX: was toggle(), now add() — re-hovering won't close it
      });

      this.renderLevel(allItems, subUl, item.id, currentLevel);
      li.appendChild(subUl);
    } else if (item.ref && item.ref !== "#") {
      li.addEventListener("click", (e) => {
        e.stopPropagation();
        console.log("Menu item data:", item);
        const winManager = new WindowManager(item);
        winManager.openWindow();
      });
    }

    return li;
  }

  showError() {
    this.content.innerHTML = '<div class="loader">Ошибка загрузки</div>';
  }

  destroy() {
    this.content.innerHTML = "";
    this.isLoaded = false;
  }
}
