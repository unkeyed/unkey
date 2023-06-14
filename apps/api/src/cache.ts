export class Cache<T> {
  private ttlSeconds: number;
  private store: Map<string, { value: T; expiry: number }>;

  constructor(opts: {
    ttlSeconds: number;
  }) {
    this.store = new Map();
    this.ttlSeconds = opts.ttlSeconds;

    setInterval(() => {
      const now = Date.now();
      for (const [key, value] of this.store.entries()) {
        if (value.expiry <= now) {
          this.store.delete(key);
        }
      }
    }, opts.ttlSeconds);
  }

  public set(key: string, value: T): void {
    const expiry = Date.now() + this.ttlSeconds * 1000;
    this.store.set(key, { value, expiry });
  }

  public get(key: string): T | null {
    const item = this.store.get(key);
    if (!item) {
      return null;
    }
    if (item.expiry <= Date.now()) {
      this.store.delete(key);
      return null;
    }
    return item.value;
  }

  public delete(key: string): boolean {
    return this.store.delete(key);
  }

  public clear(): void {
    this.store.clear();
  }
}
