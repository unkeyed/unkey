import { TieredCache } from "@/pkg/cache/tiered";
import type { Api, Database, Key } from "@/pkg/db";
import { Logger } from "@/pkg/logging";
import { Metrics } from "@/pkg/metrics";
import type { RateLimiter } from "@/pkg/ratelimit";
import type { UsageLimiter } from "@/pkg/usagelimit";
import { sha256 } from "@unkey/hash";
import { RBAC, RoleQuery } from "@unkey/rbac";
import { type Result, result } from "@unkey/result";
import type { Context } from "hono";
import { Analytics } from "../analytics";
import { CacheNamespaces } from "../global";

type VerifyKeyResult =
  | {
      valid: false;
      code: "NOT_FOUND";
      key?: never;
      api?: never;
      ratelimit?: never;
      remaining?: never;
    }
  | {
      valid: false;
      publicMessage?: string;
      code:
        | "FORBIDDEN"
        | "RATE_LIMITED"
        | "USAGE_EXCEEDED"
        | "DISABLED"
        | "INSUFFICIENT_PERMISSIONS";
      key: Key;
      api: Api;
      ratelimit?: {
        remaining: number;
        limit: number;
        reset: number;
      };
      remaining?: number;
    }
  | {
      code?: never;
      valid: true;
      key: Key;
      api: Api;
      ratelimit?: {
        remaining: number;
        limit: number;
        reset: number;
      };
      remaining?: number;
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
  private readonly rlCache: Map<string, number>;
  private readonly usageLimiter: UsageLimiter;
  private readonly analytics: Analytics;
  private readonly rateLimiter: RateLimiter;
  private readonly rbac: RBAC;

  constructor(opts: {
    cache: TieredCache<CacheNamespaces>;
    logger: Logger;
    metrics: Metrics;
    db: Database;
    rateLimiter: RateLimiter;
    usageLimiter: UsageLimiter;
    analytics: Analytics;
  }) {
    this.cache = opts.cache;
    this.logger = opts.logger;
    this.db = opts.db;
    this.metrics = opts.metrics;
    this.rateLimiter = opts.rateLimiter;
    this.usageLimiter = opts.usageLimiter;
    this.rlCache = new Map();
    this.analytics = opts.analytics;
    this.rbac = new RBAC();
  }

  public async verifyKey(
    c: Context,
    req: { key: string; apiId?: string; roleQuery?: RoleQuery },
  ): Promise<Result<VerifyKeyResult>> {
    const res = await this._verifyKey(c, req).catch(async (e) => {
      this.logger.error("Unhandled error while verifying key", {
        error: (e as Error).message,
        keyHash: await sha256(req.key),
        apiId: req.apiId,
      });
      throw e;
    });
    if (res.error) {
      this.metrics.emit("metric.key.verification", {
        valid: false,
        code: res.error.message,
      });
      return res;
    }
    // if we have identified the key, we can send the analytics event
    // otherwise, they likely sent garbage to us and we can't associate it with anything
    if (res.value.key) {
      c.executionCtx.waitUntil(
        this.analytics.ingestKeyVerification({
          workspaceId: res.value.key.workspaceId,
          apiId: res.value.api.id,
          keyId: res.value.key.id,
          time: Date.now(),
          deniedReason: res.value.code,
          ipAddress: c.req.header("True-Client-IP") ?? c.req.header("CF-Connecting-IP"),
          userAgent: c.req.header("User-Agent"),
          requestedResource: "",
          edgeRegion: "",
          // @ts-expect-error - the cf object will be there on cloudflare
          region: c.req.raw?.cf?.colo ?? "",
        }),
      );
    }

    this.metrics.emit("metric.key.verification", {
      valid: res.value.valid,
      code: res.value.code ?? "OK",
      workspaceId: res.value.key?.workspaceId,
      apiId: res.value.api?.id,
      keyId: res.value.key?.id,
    });

    return res;
  }

  /**
   * extracting this into a separate function just makes it easier to emit the analytics event
   */
  private async _verifyKey(
    c: Context,
    req: { key: string; apiId?: string; roleQuery?: RoleQuery },
  ): Promise<Result<VerifyKeyResult>> {
    const hash = await sha256(req.key);

    const data = await this.cache.withCache(c, "keyByHash", hash, async () => {
      const dbStart = performance.now();
      const dbRes = await this.db.query.keys.findFirst({
        where: (table, { and, eq, isNull }) => and(eq(table.hash, hash), isNull(table.deletedAt)),
        with: {
          roles: {
            columns: { role: true },
          },
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
      if (!dbRes) {
        return null;
      }
      if (!dbRes.keyAuth.api) {
        this.logger.error("database did not return api for key", dbRes);
      }
      return { key: dbRes, api: dbRes.keyAuth.api };
    });

    if (!data) {
      return result.success({ valid: false, code: "NOT_FOUND" });
    }
    /**
     * Enabled
     */
    const enabled = data.key.enabled;
    if (!enabled) {
      return result.success({
        key: data.key,
        api: data.api,
        valid: false,
        code: "DISABLED",
      });
    }

    if (req.apiId && data.api.id !== req.apiId) {
      return result.success({
        key: data.key,
        api: data.api,
        valid: false,
        code: "FORBIDDEN",
      });
    }

    /**
     * Expiration
     *
     * There is an issue with our zone cache, that returns dates as strings, so we need to handle that
     */
    const expires = data.key.expires ? new Date(data.key.expires).getTime() : undefined;
    if (expires) {
      if (expires < Date.now()) {
        return result.success({ valid: false, code: "NOT_FOUND" });
      }
    }

    if (data.api.ipWhitelist) {
      const ip = c.req.header("True-Client-IP") ?? c.req.header("CF-Connecting-IP");
      if (!ip) {
        return result.success({
          key: data.key,
          api: data.api,
          valid: false,
          code: "FORBIDDEN",
        });
      }
      const ipWhitelist = JSON.parse(data.api.ipWhitelist) as string[];
      if (!ipWhitelist.includes(ip)) {
        return result.success({
          key: data.key,
          api: data.api,
          valid: false,
          code: "FORBIDDEN",
        });
      }
    }

    if (req.roleQuery) {
      const roleResp = this.rbac.evaluateRoles(
        req.roleQuery,
        data.key.roles?.map((r) => r.role) ?? [],
      );
      if (roleResp.error) {
        return result.fail({ message: roleResp.error.message, code: "INTERNAL_SERVER_ERROR" });
      }
      if (!roleResp.value.valid) {
        return result.success({
          key: data.key,
          api: data.api,
          valid: false,
          code: "INSUFFICIENT_PERMISSIONS",
        });
      }
    }

    /**
     * Ratelimiting
     */
    const [pass, ratelimit] = await this.ratelimit(c, data.key);
    if (!pass) {
      return result.success({
        key: data.key,
        api: data.api,
        valid: false,
        code: "RATE_LIMITED",
        ratelimit,
      });
    }

    let remaining: number | undefined = undefined;
    if (data.key.remaining !== null) {
      const limited = await this.usageLimiter.limit({ keyId: data.key.id });
      remaining = limited.remaining;
      if (!limited.valid) {
        return result.success({
          key: data.key,
          api: data.api,
          valid: false,
          code: "USAGE_EXCEEDED",
          keyId: data.key.id,
          apiId: data.api.id,
          ownerId: data.key.ownerId ?? undefined,
          expires,
          remaining,
          ratelimit,
          isRootKey: !!data.key.forWorkspaceId,
          authorizedWorkspaceId: data.key.forWorkspaceId ?? data.key.workspaceId,
        });
      }
    }

    return result.success({
      workspaceId: data.key.workspaceId,
      key: data.key,
      api: data.api,
      valid: true,
      ownerId: data.key.ownerId ?? undefined,
      expires,
      ratelimit,
      remaining,
      isRootKey: !!data.key.forWorkspaceId,
      authorizedWorkspaceId: data.key.forWorkspaceId ?? data.key.workspaceId,
    });
  }

  /**
   * @returns [pass, ratelimit]
   */
  private async ratelimit(c: Context, key: Key): Promise<[boolean, VerifyKeyResult["ratelimit"]]> {
    if (
      !key.ratelimitType ||
      !key.ratelimitLimit ||
      !key.ratelimitRefillRate ||
      !key.ratelimitRefillInterval
    ) {
      return [true, undefined];
    }
    if (!this.rateLimiter) {
      this.logger.warn("ratelimiting is not enabled, but a key has ratelimiting enabled");
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
            limit: key.ratelimitRefillRate,
            reset,
          },
        ];
      }

      const remaining = remainingBeforeCall - 1;

      // TODO: at some point we should remove counters from older windows
      // but I'm pretty sure it's not an issue cause they take up very little memory
      // and are reset when the worker deallocates
      this.rlCache.set(keyAndWindow, cached + 1);
      const t2 = performance.now();
      const p = this.rateLimiter
        .limit({
          keyId: key.id,
          limit: key.ratelimitRefillRate,
          interval: key.ratelimitRefillInterval,
        })
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
            limit: key.ratelimitRefillRate,
            reset,
          },
        ];
      }
      const current = await p;
      return [
        current <= key.ratelimitRefillRate,
        {
          remaining: key.ratelimitRefillRate - current,
          limit: key.ratelimitRefillRate,
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
