import Redis from "ioredis";
type Identifier = string;

type KeyConfig = {
  /**
   * The maximum number of tokens that can be consumed
   * This is useful for allowing bursts of traffic
   */
  limit: number;

  /**
   *  How many tokens to refill per interval
   */
  refillRate: number;

  /**
   * How long to wait between refills in milliseconds
   */
  refillInterval: number;
};

type Key = {
  /**
   * Currently available tokens
   */
  available: number;

  /**
   * Timestamp of when the key was last refilled in milliseconds
   */
  lastRefill: number;

  /**
   * Timestamp of when we can safely delete this key from memory, in milliseconds
   */
  expires: number;
};

export type RatelimitResult = {
  pass: boolean;
  limit: number;
  remaining: number;
  reset: number;
};
export class Ratelimiter {
  private store: Map<Identifier, Key>;
  private redis: Redis;

  constructor(redisUrl: string) {
    this.store = new Map();
    this.redis = new Redis(redisUrl, { family: 6 });

    setTimeout(() => {
      const now = Date.now();
      for (const [identifier, key] of this.store.entries()) {
        if (key.expires < now) {
          this.store.delete(identifier);
        }
      }
    }, 60_000);
  }
  /**
   * This method uses a redis db to sync the ratelimit across all our servers
   * */
  public async limitGlobal(
    identifier: Identifier,
    config: KeyConfig,
    requestedTokens = 1,
  ): Promise<RatelimitResult> {
    const script = `
    local key             = KEYS[1]           -- identifier including prefixes
    local maxTokens       = tonumber(ARGV[1]) -- maximum number of tokens
    local interval        = tonumber(ARGV[2]) -- size of the window in milliseconds
    local refillRate      = tonumber(ARGV[3]) -- how many tokens are refilled after each interval
    local now             = tonumber(ARGV[4]) -- current timestamp in milliseconds
    local requestedTokens = tonumber(ARGV[5]) -- how many tokens are requested for this operation
    local remaining   = 0
    
    local bucket = redis.call("HMGET", key, "updatedAt", "tokens")
    
    if bucket[1] == false then
      -- The bucket does not exist yet, so we create it.
      remaining = maxTokens - requestedTokens
      
      redis.call("HMSET", key, "updatedAt", now, "tokens", remaining)

      return {remaining, now + interval}
    end

    local updatedAt = tonumber(bucket[1])
    local tokens = tonumber(bucket[2])

    if now >= updatedAt + interval then
      local numberOfRefills = math.floor((now - updatedAt)/interval)

      if tokens <= 0 then 
        -- No more tokens were left before the refill.
        remaining = math.min(maxTokens, numberOfRefills * refillRate) - requestedTokens
      else
        remaining = math.min(maxTokens, tokens + numberOfRefills * refillRate) - requestedTokens
      end

      local lastRefill = updatedAt + numberOfRefills * interval

      redis.call("HMSET", key, "updatedAt", lastRefill, "tokens", remaining)
      return {remaining, lastRefill + interval}
    end
  
  remaining = tokens - requestedTokens
  redis.call("HSET", key, "tokens", remaining)
  return {remaining, updatedAt + interval}


   `;

    const now = Date.now();

    const res = await this.redis.eval(
      script,
      1,
      identifier,
      config.limit,
      config.refillInterval,
      config.refillRate,
      now,
      requestedTokens,
    );
    const [remaining, reset] = res as [number, number];

    return {
      pass: remaining > 0,
      limit: config.limit,
      remaining,
      reset,
    };
  }

  // Quick and dirty in-memory ratelimiting
  public limitLocal(
    identifier: Identifier,
    config: KeyConfig,
    requestedTokens = 1,
  ): RatelimitResult {
    const now = Date.now();
    let key = this.store.get(identifier);
    if (!key) {
      key = {
        available: config.limit - 1,
        lastRefill: now,
        expires: now + config.refillInterval * 2,
      };
      this.store.set(identifier, key);
      return {
        pass: true,
        limit: config.limit,
        remaining: key.available,
        reset: now + config.refillInterval,
      };
    }

    /**
     * Refill the bucket
     */
    const elapsed = now - key.lastRefill;
    const tokensToAdd = Math.floor(elapsed * (config.refillRate / config.refillInterval));
    if (tokensToAdd > 0) {
      key.available = Math.min(config.limit, key.available + tokensToAdd);
      key.lastRefill = now;
      key.expires = now + config.refillInterval * 2;
    }

    /**
     * Check if enough tokens are available
     */

    if (requestedTokens > key.available) {
      return {
        pass: false,
        limit: config.limit,
        remaining: key.available,
        reset: now + config.refillInterval,
      };
    }
    key.available -= requestedTokens;
    this.store.set(identifier, key);
    return {
      pass: true,
      limit: config.limit,
      remaining: key.available,
      reset: now + config.refillInterval,
    };
  }
}
