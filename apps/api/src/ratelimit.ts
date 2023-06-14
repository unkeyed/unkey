type ApiKey = string;
type Window = number;
type Identifier = `${ApiKey}:${Window}`;

type Key = {
  limit: number;
  // milliseconds
  duration: number;

  // unix milli timestamp
  resetAt: number;

  // usage in this window
  current: number;
};

export class Ratelimiter {
  private store: Map<Identifier, Key>;

  constructor() {
    this.store = new Map();

    setInterval(() => {
      const now = Date.now();
      for (const [key, value] of this.store.entries()) {
        if (value.resetAt <= now) {
          this.store.delete(key);
        }
      }
    }, 10_000);
  }

  public limit(req: { key: ApiKey; duration: number; limit: number }): {
    pass: boolean;
    limit: number;
    remaining: number;
    reset: number;
  } {
    const now = Date.now();
    const window = now % req.duration;
    const identifier: Identifier = `${req.key}:${window}`;
    let key = this.store.get(identifier);
    if (!key) {
      key = {
        limit: req.limit,
        duration: req.duration,
        resetAt: now + req.duration,
        current: 0,
      };
      this.store.set(identifier, key);
    }

    if (key.current >= key.limit) {
      return {
        pass: false,
        limit: key.limit,
        remaining: 0,
        reset: key.resetAt,
      };
    }
    key.current++;
    this.store.set(identifier, key);
    return {
      pass: true,
      limit: key.limit,
      remaining: key.limit - key.current,
      reset: key.resetAt,
    };
  }
}
