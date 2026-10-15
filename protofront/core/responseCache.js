/* TTL response cache (single responsibility: storage + expiry only,
 * no transport, no URL knowledge). */
class ResponseCache {
  constructor(ttlMs = 5 * 60 * 1000) {
    this.ttlMs = ttlMs;
    this.entries = new Map();
    this.timestamps = new Map();
  }

  has(key) {
    return this.entries.has(key);
  }

  get(key) {
    return this.entries.get(key);
  }

  set(key, value) {
    this.entries.set(key, value);
    this.timestamps.set(key, Date.now());
  }

  isFresh(key) {
    if (!this.entries.has(key)) return false;
    const ts = this.timestamps.get(key);
    if (!ts) return false;
    return Date.now() - ts < this.ttlMs;
  }

  clear(key = null) {
    if (key) {
      this.entries.delete(key);
      this.timestamps.delete(key);
    } else {
      this.entries.clear();
      this.timestamps.clear();
    }
  }
}
