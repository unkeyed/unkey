import { type Api, type Database, type Key } from "@unkey/db";
import { type Result, result } from "@unkey/result";
import type { Context } from "hono";
import type { Cache } from "../cache/interface";
import { KeyHash } from "../context/global";
import { sha256 } from "../hash/sha256";
import { Logger } from "../logging";
import { Metrics } from "../metrics";
import { withCache } from "../cache/with_cache";

type VerifyKeyResult =
  | {
    valid: false;
    code: string;
  }
  | {
    apiId?: string;
    valid: true;
    ownerId?: string;
    meta?: Record<string, unknown>;
    expires?: number;
    remaining?: number;
    ratelimit?: {
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

  constructor(opts: {
    verificationCache: Cache<KeyHash, { key: Key; api: Api } | null>;
    logger: Logger;
    metrics: Metrics;
    db: Database;
  }) {
    this.verificationCache = opts.verificationCache;
    this.logger = opts.logger;
    this.db = opts.db;
    this.metrics = opts.metrics;
  }

  public async verifyKey(
    c: Context,
    req: { key: string; apiId?: string },
  ): Promise<Result<VerifyKeyResult>> {
    const hash = await sha256(req.key);



    const data = await withCache(c, this.verificationCache, async (h: KeyHash) => {
      const dbStart = performance.now();
      const dbRes = await this.db.query.keys.findFirst({
        where: (table, { eq }) => eq(table.hash, h),
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

    // TODO: Ratelimit
    // TODO: Expiration
    // TODO: Remaining

    if (req.apiId && data.api.id !== req.apiId) {
      return result.success({ valid: false, code: "FORBIDDEN" });
    }

    return result.success({
      apiId: data.api.id,
      valid: true,
      ownerId: data.key.ownerId ?? undefined,
      meta: data.key.meta ? (JSON.parse(data.key.meta) as Record<string, unknown>) : undefined,
      expires: data.key.expires?.getTime() ?? undefined,
      remaining: data.key.remainingRequests ?? undefined,
      ratelimit: undefined, // TODO
      isRootKey: !!data.key.forWorkspaceId,
      authorizedWorkspaceId: data.key.forWorkspaceId ?? data.key.workspaceId,
    });
  }
}
