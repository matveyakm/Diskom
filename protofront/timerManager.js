// Класс для обновления времени
class TimeManager {
  constructor(timeDisplayElement) {
    this.timeDisplay = timeDisplayElement;
    this.interval = null;
  }

  // Запуск обновления времени
  start() {
    this.update();
    this.interval = setInterval(() => this.update(), 1000);
  }

  // Обновление времени
  update() {
    const now = new Date();
    this.timeDisplay.innerText = now.toLocaleTimeString([], {
      hour: "2-digit",
      minute: "2-digit",
    });
  }

  // Остановка обновления
  stop() {
    if (this.interval) {
      clearInterval(this.interval);
      this.interval = null;
    }
  }
}
