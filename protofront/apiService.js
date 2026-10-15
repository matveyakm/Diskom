// API service: endpoint-bound CRUD over shared transport + cache.
// Transport (HttpClient) and storage (ResponseCache) are injected
// capabilities; this class owns URL shaping and cache policy only.
class ApiService {
  constructor(options, { http = null, cache = null } = {}) {
    this.endPoint = options.ref;
    this.keys = options.keys || {};
    this.baseUrl = options.baseUrl || window.location.origin;
    this.useCache = options.useCache !== false; // Default true
    this.http =
      http ||
      new HttpClient({ baseUrl: this.baseUrl, timeout: options.timeout || 30000 });
    this.cache = cache || ApiService.sharedCache;
  }

  buildUrl() {
    const url = new URL(this.endPoint, this.baseUrl);
    Object.keys(this.keys).forEach((key) => {
      if (this.keys[key] !== undefined && this.keys[key] !== null) {
        url.searchParams.append(key, this.keys[key]);
      }
    });
    return url.toString();
  }

  getCacheKey() {
    return this.buildUrl();
  }

  static clearCache(url = null) {
    ApiService.sharedCache.clear(url);
  }

  async getResponse() {
    const cacheKey = this.getCacheKey();

    if (this.useCache && this.cache.isFresh(cacheKey)) {
      console.log(`Returning cached data for: ${cacheKey}`);
      return this.cache.get(cacheKey);
    }

    try {
      const data = await this.http.get(this.buildUrl());
      if (this.useCache) {
        this.cache.set(cacheKey, data);
      }
      return data;
    } catch (error) {
      console.error("ApiService error:", error);
      if (this.cache.has(cacheKey)) {
        console.warn("Returning stale cache due to error");
        return this.cache.get(cacheKey);
      }
      throw error;
    }
  }

  async postResponse(body) {
    return this.mutate("post", body);
  }

  async putResponse(body) {
    return this.mutate("put", body);
  }

  async patchResponse(body) {
    return this.mutate("patch", body);
  }

  async mutate(method, body) {
    try {
      const data = await this.http[method](this.buildUrl(), body);
      ApiService.clearCache(this.getCacheKey());
      return data;
    } catch (error) {
      console.error(`ApiService ${method.toUpperCase()} error:`, error);
      throw error;
    }
  }

  async deleteResponse(id = null) {
    try {
      const url = this.buildUrl();
      const finalUrl = id ? `${url}/${id}` : url;
      const data = await this.http.delete(finalUrl);
      ApiService.clearCache(this.getCacheKey());
      return data;
    } catch (error) {
      console.error("ApiService DELETE error:", error);
      throw error;
    }
  }

  async refresh() {
    const cacheKey = this.getCacheKey();
    ApiService.clearCache(cacheKey);
    return this.getResponse();
  }

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

ApiService.sharedCache = new ResponseCache();
ApiService.cacheTTL = 5 * 60 * 1000;
