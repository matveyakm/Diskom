/* Bare HTTP transport (single responsibility: fetch with timeout,
 * no caching, no endpoint knowledge). Depends on global fetch. */
class HttpClient {
  constructor({ baseUrl = window.location.origin, timeout = 30000 } = {}) {
    this.baseUrl = baseUrl;
    this.timeout = timeout;
  }

  async request(method, url, body = undefined) {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeout);
    try {
      const response = await fetch(url, {
        method,
        signal: controller.signal,
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
        },
        ...(body !== undefined ? { body: JSON.stringify(body) } : {}),
      });
      clearTimeout(timeoutId);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }
      return response.json();
    } catch (error) {
      clearTimeout(timeoutId);
      if (error.name === "AbortError") {
        throw new Error(`Request timeout after ${this.timeout}ms`);
      }
      throw error;
    }
  }

  get(url) {
    return this.request("GET", url);
  }

  post(url, body) {
    return this.request("POST", url, body);
  }

  put(url, body) {
    return this.request("PUT", url, body);
  }

  patch(url, body) {
    return this.request("PATCH", url, body);
  }

  delete(url) {
    return this.request("DELETE", url);
  }
}
