class StartMenuApp {
  constructor() {
    this.startButton = null;
    this.menuManager = null;
    this.timeManager = null;
    this.apiService = MenuApiService;

    this._boundHandleStartButtonClick = (e) => this.handleStartButtonClick(e);
    this._boundHandleOutsideClick = () => this.handleOutsideClick();
    this._boundHandleMenuClick = (e) => this.handleMenuClick(e);
  }

  init() {
    const startButton = document.getElementById("start-button");
    const startMenu = document.getElementById("start-menu");
    const menuContent = document.getElementById("menu-content");
    const timeDisplay = document.getElementById("time");

    if (!startButton || !startMenu || !menuContent || !timeDisplay) {
      console.error("Не удалось найти необходимые DOM элементы");
      return;
    }

    this.startButton = startButton;
    this.menuManager = new MenuManager(startMenu, menuContent);
    this.timeManager = new TimeManager(timeDisplay);

    this.initTimeUpdates();
    this.initEventListeners();
  }

  initTimeUpdates() {
    this.timeManager.start();
  }

  async handleStartButtonClick(e) {
    e.stopPropagation();
    this.menuManager.toggle();

    if (this.menuManager.isActive()) {
      await this.menuManager.loadAndRender(() =>
        this.apiService.fetchMenuData(),
      );
    }
  }

  handleOutsideClick() {
    this.menuManager.close();
  }

  handleMenuClick(e) {
    e.stopPropagation();
  }

  initEventListeners() {
    this.startButton.addEventListener(
      "click",
      this._boundHandleStartButtonClick,
    );
    document.addEventListener("click", this._boundHandleOutsideClick);

    const startMenu = document.getElementById("start-menu");
    if (startMenu) {
      startMenu.addEventListener("click", this._boundHandleMenuClick);
    }
  }

  destroy() {
    this.timeManager.stop();
    this.menuManager.destroy();

    this.startButton.removeEventListener(
      "click",
      this._boundHandleStartButtonClick,
    );
    document.removeEventListener("click", this._boundHandleOutsideClick);

    const startMenu = document.getElementById("start-menu");
    if (startMenu) {
      startMenu.removeEventListener("click", this._boundHandleMenuClick);
    }
  }
}
