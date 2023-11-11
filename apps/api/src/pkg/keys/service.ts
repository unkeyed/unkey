import { type Api, type Database, type Key } from "@unkey/db";
import { type Result, result } from "@unkey/result";
import type { Context } from "hono";
import type { Cache } from "../cache/interface";
import { KeyHash } from "../context/global";
import { sha256 } from "../hash/sha256";
import { Logger } from "../logging";
import { Metrics } from "../metrics";
import { withCache } from "../cache/with_cache";
import { Env } from "../env";

type VerifyKeyResult =
  | {
    valid: false;
    code: string;
    ratelimit?: {
      remaining: number
      limit: number;
      reset: number;
    };
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
      remaining: number
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
  private readonly verificationCache: Cache<KeyHash, { key: Key; api: Api } | null>;
  private readonly logger: Logger;
  private readonly metrics: Metrics
  private readonly db: Database;
  private readonly rl: DurableObjectNamespace
  private readonly rlCache: Map<string, { current: number, reset: number }>

  constructor(opts: {
    verificationCache: Cache<KeyHash, { key: Key; api: Api } | null>;
    logger: Logger;
    metrics: Metrics;
    db: Database;
    rl: DurableObjectNamespace
  }) {
    this.verificationCache = opts.verificationCache;
    this.logger = opts.logger;
    this.db = opts.db;
    this.metrics = opts.metrics;
    this.rl = opts.rl
    this.rlCache = new Map()
  }

  public async verifyKey(
    c: Context,
    req: { key: string; apiId?: string },
  ): Promise<Result<VerifyKeyResult>> {
    const hash = await sha256(req.key);



    const data = await withCache(c, this.verificationCache, async (h: KeyHash) => {
      const dbStart = performance.now();
      const dbRes = await this.db.query.keys.findFirst({
        where: (table, { and, eq, isNull }) => and(eq(table.hash, h), isNull(table.deletedAt)),
        with: {
          keyAuth: {
            with: {
              api: true,
            },
          },
        },
      })
      this.metrics.emit("metric.db.read", {
        query: "getKeyAndApiByHash",
        latency: performance.now() - dbStart
      })
      return dbRes ? { key: dbRes, api: dbRes.keyAuth.api } : null;
    })(hash)


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

    let ratelimit: VerifyKeyResult["ratelimit"] = undefined
    if (data.key.ratelimitType && data.key.ratelimitRefillInterval && data.key.ratelimitLimit) {
      const now = Date.now()
      const window = Math.floor(now / data.key.ratelimitRefillInterval)
      const reset = (window + 1) * data.key.ratelimitRefillInterval

      const rlKeyWindow = [data.key.id, window].join(":")
      const t1 = performance.now()
      let rl = this.rlCache.get(rlKeyWindow)
      if (rl && rl.reset > Date.now()) {
        rl = undefined
        this.rlCache.delete(rlKeyWindow)
      }
      this.metrics.emit("metric.ratelimit", {
        hit: !!rl,
        latency: performance.now() - t1,
        keyId: data.key.id,
        tier: "memory"
      })
      this.rlCache.set(rlKeyWindow, { reset, current: rl?.current ? rl.current + 1 : 1 })
      ratelimit = {
        remaining: data.key.ratelimitLimit - (rl?.current ? rl.current + 1 : 1),
        limit: data.key.ratelimitLimit,
        reset
      }



      if (rl && rl.current >= data.key.ratelimitLimit) {

        return result.success({
          valid: false,
          code: "RATELIMITED",
          ratelimit: {
            remaining: data.key.ratelimitLimit - rl.current,
            limit: data.key.ratelimitLimit,
            reset: rl.reset
          }
        });
      }




      const obj = this.rl.get(this.rl.idFromName(rlKeyWindow))
      const t2 = performance.now()
      const p = obj.fetch(new URL("https://unkey.app.com"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          reset
        })
      }).then(async res => await res.json() as { current: number }).then(res => {
        this.metrics.emit("metric.ratelimit", {
          hit: res.current > 1, // 1 would mean the object did not exist and was just created and incremented
          latency: performance.now() - t2,
          keyId: data.key.id,
          tier: "durable"
        })
        this.rlCache.set(rlKeyWindow, { current: res.current, reset })
        rl = {
          current: res.current,
          reset,
        }
        this.rlCache.set(rlKeyWindow, rl)
        ratelimit = {
          remaining: data.key.ratelimitLimit! - res.current,
          limit: data.key.ratelimitLimit!,
          reset
        }
      })

      if (data.key.ratelimitType === "consistent") {
        await p
      } else {
        c.executionCtx.waitUntil(p)
      }

      if (rl!.current > data.key.ratelimitLimit) {
        return result.success({
          valid: false,
          code: "RATELIMITED",
          ratelimit
        });
      }

    }





    // TODO: Ratelimit
    // TODO: Remaining


    return result.success({
      keyId: data.key.id,
      apiId: data.api.id,
      valid: true,
      ownerId: data.key.ownerId ?? undefined,
      meta: data.key.meta ? (JSON.parse(data.key.meta) as Record<string, unknown>) : undefined,
      expires: data.key.expires?.getTime() ?? undefined,
      remaining: data.key.remainingRequests ?? undefined,
      ratelimit,
      isRootKey: !!data.key.forWorkspaceId,
      authorizedWorkspaceId: data.key.forWorkspaceId ?? data.key.workspaceId,
    });
  }
}
