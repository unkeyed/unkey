import { sha256 } from "../hash/sha256";
import type { Cache } from "../cache/interface";
import { type Key, type Api, type Database } from "@unkey/db";
import { result, type Result } from "@unkey/result";
import { Logger } from "../logging";
import { KeyHash } from "../context/global";
import type { Context } from "hono";

type VerifyKeyResult = {
  valid: false,
  code: string
} | {
  apiId?: string
  valid: true
  ownerId?: string
  meta?: Record<string, unknown>
  expires?: number
  remaining?: number
  ratelimit?: {
    limit: number
    reset: number
  }


  isRootKey?: boolean
  /**
 * the workspace of the user, even if this is a root key
  */
  authorizedWorkspaceId: string
}

export class KeyService {

  private readonly verificationCache: Cache<KeyHash, { key: Key, api: Api }>
  private readonly logger: Logger
  private readonly db: Database


  constructor(opts: { verificationCache: Cache<KeyHash, { key: Key, api: Api }>, logger: Logger, db: Database }) {
    this.verificationCache = opts.verificationCache
    this.logger = opts.logger
    this.db = opts.db
  }

  public async verifyKey(c: Context, req: { key: string, apiId?: string }): Promise<Result<VerifyKeyResult>> {
    const hash = await sha256(req.key)
    let data = await this.verificationCache.get(c, hash)
    if (data === null) {
      return result.success({ valid: false, code: "NOT_FOUND" })
    }



    if (!data) {
      const dbRes = await this.db.query.keys.findFirst({
        where: (table, { eq }) => eq(table.hash, hash),
        with: {
          keyAuth: {
            with: {
              api: true
            }
          }
        }
      })
      if (!dbRes) {
        return result.success({ valid: false, code: "NOT_FOUND" })
      }
      data = { key: dbRes, api: dbRes.keyAuth.api }
      await this.verificationCache.set(c, hash, data)
    }


    // TODO: Ratelimit
    // TODO: Expiration
    // TODO: Remaining


    if (req.apiId && data.api.id !== req.apiId) {
      return result.success({ valid: false, code: "FORBIDDEN" })
    }

    return result.success({
      apiId: data.api.id,
      valid: true,
      ownerId: data.key.ownerId ?? undefined,
      meta: data.key.meta ? JSON.parse(data.key.meta) as Record<string, unknown> : undefined,
      expires: data.key.expires?.getTime() ?? undefined,
      remaining: data.key.remainingRequests ?? undefined,
      ratelimit: undefined, // TODO
      isRootKey: !!data.key.forWorkspaceId,
      authorizedWorkspaceId: data.key.forWorkspaceId ?? data.key.workspaceId
    })




  }
}
