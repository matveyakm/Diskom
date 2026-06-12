// API сервис для получения данных
class ApiService {
  constructor(options) {
    this.endPoint = options.ref;
    this.keys = options.keys || {};
    this.baseUrl = options.baseUrl || window.location.origin;
    this.useCache = options.useCache !== false; // Default true
    this.cacheTTL = options.cacheTTL || ApiService.cacheTTL;
    this.timeout = options.timeout || 30000; // 30 seconds timeout
  }

  // Build URL with query parameters
  buildUrl() {
    const url = new URL(this.endPoint, this.baseUrl);
    Object.keys(this.keys).forEach((key) => {
      if (this.keys[key] !== undefined && this.keys[key] !== null) {
        url.searchParams.append(key, this.keys[key]);
      }
    });
    return url.toString();
  }

  // Get cache key
  getCacheKey() {
    return this.buildUrl();
  }

  // Check if cache is valid
  isCacheValid(cacheKey) {
    if (!this.useCache) return false;
    if (!ApiService.cache.has(cacheKey)) return false;

    const timestamp = ApiService.cacheTimestamps.get(cacheKey);
    if (!timestamp) return false;

    return Date.now() - timestamp < this.cacheTTL;
  }

  // Clear cache for specific URL or all
  static clearCache(url = null) {
    if (url) {
      ApiService.cache.delete(url);
      ApiService.cacheTimestamps.delete(url);
    } else {
      ApiService.cache.clear();
      ApiService.cacheTimestamps.clear();
    }
  }

  // Get response with timeout
  async fetchWithTimeout(url, options) {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeout);

    try {
      const response = await fetch(url, {
        ...options,
        signal: controller.signal,
      });
      clearTimeout(timeoutId);
      return response;
    } catch (error) {
      clearTimeout(timeoutId);
      if (error.name === "AbortError") {
        throw new Error(`Request timeout after ${this.timeout}ms`);
      }
      throw error;
    }
  }

  // Main method to get response
  async getResponse() {
    const cacheKey = this.getCacheKey();

    // Check cache
    if (this.isCacheValid(cacheKey)) {
      console.log(`Returning cached data for: ${cacheKey}`);
      return ApiService.cache.get(cacheKey);
    }

    try {
      const url = this.buildUrl();
      console.log(`Fetching data from: ${url}`);

      const response = await this.fetchWithTimeout(url, {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
        },
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const data = await response.json();

      // Store in cache
      if (this.useCache) {
        ApiService.cache.set(cacheKey, data);
        ApiService.cacheTimestamps.set(cacheKey, Date.now());
      }

      return data;
    } catch (error) {
      console.error("ApiService error:", error);

      // Return stale cache if available (offline fallback)
      if (ApiService.cache.has(cacheKey)) {
        console.warn("Returning stale cache due to error");
        return ApiService.cache.get(cacheKey);
      }

      throw error;
    }
  }

  // POST method for creating data
  async postResponse(body) {
    try {
      const url = this.buildUrl();
      const response = await fetch(url, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
        },
        body: JSON.stringify(body),
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const data = await response.json();

      // Clear cache for this endpoint after POST
      ApiService.clearCache(this.getCacheKey());

      return data;
    } catch (error) {
      console.error("ApiService POST error:", error);
      throw error;
    }
  }

  // PUT method for updating data
  async putResponse(body) {
    try {
      const url = this.buildUrl();
      const response = await fetch(url, {
        method: "PUT",
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
        },
        body: JSON.stringify(body),
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const data = await response.json();

      // Clear cache for this endpoint after PUT
      ApiService.clearCache(this.getCacheKey());

      return data;
    } catch (error) {
      console.error("ApiService PUT error:", error);
      throw error;
    }
  }

  // DELETE method
  async deleteResponse(id = null) {
    try {
      const url = this.buildUrl();
      const finalUrl = id ? `${url}/${id}` : url;

      const response = await fetch(finalUrl, {
        method: "DELETE",
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
        },
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const data = await response.json();

      // Clear cache for this endpoint after DELETE
      ApiService.clearCache(this.getCacheKey());

      return data;
    } catch (error) {
      console.error("ApiService DELETE error:", error);
      throw error;
    }
  }

  // PATCH method for partial updates
  async patchResponse(body) {
    try {
      const url = this.buildUrl();
      const response = await fetch(url, {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
        },
        body: JSON.stringify(body),
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const data = await response.json();

      // Clear cache for this endpoint after PATCH
      ApiService.clearCache(this.getCacheKey());

      return data;
    } catch (error) {
      console.error("ApiService PATCH error:", error);
      throw error;
    }
  }

  // Method to force refresh (ignore cache)
  async refresh() {
    const cacheKey = this.getCacheKey();
    ApiService.clearCache(cacheKey);
    return this.getResponse();
  }

  // Batch request multiple endpoints
  static async batchRequests(requests) {
    try {
      const promises = requests.map((req) => {
        const service = new ApiService({
          ref: req.endPoint,
          keys: req.keys || {},
        });
        return service.getResponse();
      });

      return await Promise.all(promises);
    } catch (error) {
      console.error("Batch requests error:", error);
      throw error;
    }
  }
}

ApiService.cache = new Map();
ApiService.cacheTTL = 5 * 60 * 1000;
ApiService.cacheTimestamps = new Map();
