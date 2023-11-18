import { type Database, type Key } from "@unkey/db";
import { type Result, result } from "@unkey/result";
import type { Context } from "hono";
import { sha256 } from "@/pkg/hash/sha256";
import { Logger } from "@/pkg/logging";
import { Metrics } from "@/pkg/metrics";
import { durableUsageLimit } from "../usagelimit";
import { TieredCache } from "@/pkg/cache/tiered";
import { CacheNamespaces } from "../global";

type VerifyKeyResult =
  | {
      valid: false;
      code: string;
      ratelimit?: {
        remaining: number;
        limit: number;
        reset: number;
      };
      remaining?: number;
    }
  | {
      keyId: string;
      apiId?: string;
      valid: true;
      ownerId?: string;
      meta?: Record<string, unknown>;
      expires?: number;
      remaining?: number;
      ratelimit?: {
        remaining: number;
        limit: number;
        reset: number;
      };

      isRootKey?: boolean;
      /**
       * the workspace of the user, even if this is a root key
       */
      authorizedWorkspaceId: string;
    };

export class KeyService {
  private readonly cache: TieredCache<CacheNamespaces>;
  private readonly logger: Logger;
  private readonly metrics: Metrics;
  private readonly db: Database;
  private readonly rl: DurableObjectNamespace;
  private readonly ul: DurableObjectNamespace;
  private readonly rlCache: Map<string, number>;

  constructor(opts: {
    cache: TieredCache<CacheNamespaces>;
    logger: Logger;
    metrics: Metrics;
    db: Database;
    ul: DurableObjectNamespace;
    rl: DurableObjectNamespace;
  }) {
    this.cache = opts.cache;
    this.logger = opts.logger;
    this.db = opts.db;
    this.metrics = opts.metrics;
    this.rl = opts.rl;
    this.ul = opts.ul;
    this.rlCache = new Map();
  }

  public async verifyKey(
    c: Context,
    req: { key: string; apiId?: string },
  ): Promise<Result<VerifyKeyResult>> {
    const hash = await sha256(req.key);

    const data = await this.cache.withCache(c, "keyByHash", hash, async () => {
      const dbStart = performance.now();
      const dbRes = await this.db.query.keys.findFirst({
        where: (table, { and, eq, isNull }) => and(eq(table.hash, hash), isNull(table.deletedAt)),
        with: {
          keyAuth: {
            with: {
              api: true,
            },
          },
        },
      });
      this.metrics.emit("metric.db.read", {
        query: "getKeyAndApiByHash",
        latency: performance.now() - dbStart,
      });
      return dbRes ? { key: dbRes, api: dbRes.keyAuth.api } : null;
    });

    if (!data) {
      return result.success({ valid: false, code: "NOT_FOUND" });
    }

    if (req.apiId && data.api.id !== req.apiId) {
      return result.success({ valid: false, code: "FORBIDDEN" });
    }

    /**
     * Expiration
     */
    if (data.key.expires !== null && data.key.expires.getTime() < Date.now()) {
      return result.success({ valid: false, code: "NOT_FOUND" });
    }

    /**
     * Ratelimiting
     */

    const [pass, ratelimit] = await this.ratelimit(c, data.key);
    if (!pass) {
      return result.success({
        valid: false,
        code: "RATELIMITED",
        ratelimit,
      });
    }

    const limited = await durableUsageLimit(this.ul, data.key);
    if (!limited.valid) {
      return result.success({
        valid: false,
        code: "USAGE_EXCEEDED",
        keyId: data.key.id,
        apiId: data.api.id,
        ownerId: data.key.ownerId ?? undefined,
        meta: data.key.meta ? (JSON.parse(data.key.meta) as Record<string, unknown>) : undefined,
        expires: data.key.expires?.getTime() ?? undefined,
        remaining: limited.remaining,
        ratelimit,
        isRootKey: !!data.key.forWorkspaceId,
        authorizedWorkspaceId: data.key.forWorkspaceId ?? data.key.workspaceId,
      });
    }

    return result.success({
      keyId: data.key.id,
      apiId: data.api.id,
      valid: true,
      ownerId: data.key.ownerId ?? undefined,
      meta: data.key.meta ? (JSON.parse(data.key.meta) as Record<string, unknown>) : undefined,
      expires: data.key.expires?.getTime() ?? undefined,
      remaining: limited.remaining,
      ratelimit,
      isRootKey: !!data.key.forWorkspaceId,
      authorizedWorkspaceId: data.key.forWorkspaceId ?? data.key.workspaceId,
    });
  }

  private async ratelimit(c: Context, key: Key): Promise<[boolean, VerifyKeyResult["ratelimit"]]> {
    if (
      !key.ratelimitType ||
      !key.ratelimitLimit ||
      !key.ratelimitRefillRate ||
      !key.ratelimitRefillInterval
    ) {
      return [true, undefined];
    }

    const ratelimitStart = performance.now();
    try {
      const now = Date.now();
      const window = Math.floor(now / key.ratelimitRefillInterval);
      const reset = (window + 1) * key.ratelimitRefillInterval;

      const keyAndWindow = [key.id, window].join(":");
      const t1 = performance.now();
      const cached = this.rlCache.get(keyAndWindow) ?? 0;
      this.metrics.emit("metric.ratelimit", {
        latency: performance.now() - t1,
        keyId: key.id,
        tier: "memory",
      });

      const remainingBeforeCall = key.ratelimitLimit - cached;
      if (remainingBeforeCall <= 0) {
        return [
          false,
          {
            remaining: 0,
            limit: key.ratelimitLimit,
            reset,
          },
        ];
      }

      const remaining = remainingBeforeCall - 1;

      // TODO: at some point we should remove counters from older windows
      // but I'm pretty sure it's not an issue cause they take up very little memory
      // and are reset when the worker deallocates
      this.rlCache.set(keyAndWindow, cached + 1);

      const obj = this.rl.get(this.rl.idFromName(keyAndWindow));
      const t2 = performance.now();
      const p = obj
        .fetch("https://unkey.app.com", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            reset,
          }),
        })
        .then(async (res) => (await res.json()) as { current: number })
        .then(({ current }) => {
          this.rlCache.set(keyAndWindow, current);
          this.metrics.emit("metric.ratelimit", {
            latency: performance.now() - t2,
            keyId: key.id,
            tier: "durable",
          });
          return current;
        });

      if (key.ratelimitType === "fast") {
        c.executionCtx.waitUntil(p);
        return [
          true,
          {
            remaining,
            limit: key.ratelimitLimit,
            reset,
          },
        ];
      }
      const current = await p;
      return [
        current <= key.ratelimitLimit,
        {
          remaining: key.ratelimitLimit - current,
          limit: key.ratelimitLimit,
          reset,
        },
      ];
    } catch (e: unknown) {
      const err = e as Error;
      this.logger.error("ratelimiting failed", { error: err.message, ...err });

      return [false, undefined];
    } finally {
      this.metrics.emit("metric.ratelimit", {
        latency: performance.now() - ratelimitStart,
        keyId: key.id,
        tier: "total",
      });
    }
  }
}
