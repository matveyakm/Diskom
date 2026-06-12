// API сервис для получения данных
class MenuApiService {
  static async fetchMenuData(params = { limit: 100 }) {
    // Проверяем кэш
    if (this.cache && this.cacheExpiry && Date.now() < this.cacheExpiry) {
      console.log("Возвращаем закэшированные данные меню");
      return this.cache;
    }

    try {
      // Строим URL с параметрами
      const url = new URL(
        "/api/system_object_directory",
        window.location.origin,
      );
      Object.keys(params).forEach((key) => {
        url.searchParams.append(key, params[key]);
      });

      const response = await fetch(url, {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
        },
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const responseData = await response.json();
      let data = responseData.data;

      // Sort by OutputOrder
      if (data && Array.isArray(data)) {
        data = data.sort(
          (a, b) => (a.output_order || 0) - (b.output_order || 0),
        );
      }

      // Сохраняем в кэш
      this.cache = data;
      this.cacheExpiry = Date.now() + this.cacheTTL;

      return data;
    } catch (error) {
      console.error("MenuApiService error:", error);

      // Если есть кэш, возвращаем его даже если просрочен
      if (this.cache) {
        console.warn("Возвращаем просроченный кэш из-за ошибки сети");
        return this.cache;
      }

      throw error;
    }
  }

  // Очистка кэша
  static clearCache() {
    this.cache = null;
    this.cacheExpiry = null;
  }
}

MenuApiService.cache = null;
MenuApiService.cacheExpiry = null;
MenuApiService.cacheTTL = 5 * 60 * 1000;
