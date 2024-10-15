import type { Cache } from "@/pkg/cache";
import type { Api, Database, Key, Ratelimit } from "@/pkg/db";
import type { Context } from "@/pkg/hono/app";
import type { Metrics } from "@/pkg/metrics";
import type { RateLimiter } from "@/pkg/ratelimit";
import type { UsageLimiter } from "@/pkg/usagelimit";
import { BaseError, Err, FetchError, Ok, type Result, SchemaError } from "@unkey/error";
import { sha256 } from "@unkey/hash";
import type { PermissionQuery, RBAC } from "@unkey/rbac";
import type { Logger } from "@unkey/worker-logging";
import { retry } from "../util/retry";

export class DisabledWorkspaceError extends BaseError<{ workspaceId: string }> {
  public readonly retry = false;
  public readonly name = DisabledWorkspaceError.name;
  constructor(workspaceId: string) {
    super({
      message: "workspace is disabled",
      context: {
        workspaceId,
      },
    });
  }
}

export class MissingRatelimitError extends BaseError<{ name: string }> {
  public readonly retry = false;
  public readonly name = MissingRatelimitError.name;
  constructor(ratelimitName: string, message: string) {
    super({
      message,
      context: {
        name: ratelimitName,
      },
    });
  }
}

type NotFoundResponse = {
  valid: false;
  code: "NOT_FOUND";
  key?: never;
  identity?: never;
  api?: never;
  ratelimit?: never;
  remaining?: never;
};

type InvalidResponse = {
  valid: false;
  publicMessage?: string;
  code:
    | "FORBIDDEN"
    | "RATE_LIMITED"
    | "EXPIRED"
    | "USAGE_EXCEEDED"
    | "DISABLED"
    | "INSUFFICIENT_PERMISSIONS";
  key: Key;
  identity: { id: string; externalId: string; meta: Record<string, unknown> | null } | null;
  api: Api;
  ratelimit?: {
    remaining: number;
    limit: number;
    reset: number;
  };
  remaining?: number;
  permissions: string[];
  message?: string;
};

type ValidResponse = {
  code?: never;
  valid: true;
  key: Key;
  identity: { id: string; externalId: string; meta: Record<string, unknown> | null } | null;
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
  permissions: string[];
};
type VerifyKeyResult = NotFoundResponse | InvalidResponse | ValidResponse;

type RatelimitRequest = {
  identity: string;
  name: string;
  cost?: number;
  limit?: number;
  duration?: number;
};

export class KeyService {
  private readonly cache: Cache;
  private readonly logger: Logger;
  private readonly metrics: Metrics;
  private readonly db: { primary: Database; readonly: Database };
  private readonly usageLimiter: UsageLimiter;
  private readonly rateLimiter: RateLimiter;
  private readonly rbac: RBAC;
  private readonly hashCache = new Map<string, string>();

  constructor(opts: {
    cache: Cache;
    logger: Logger;
    metrics: Metrics;
    db: { primary: Database; readonly: Database };
    rateLimiter: RateLimiter;
    usageLimiter: UsageLimiter;
    rbac: RBAC;
  }) {
    this.cache = opts.cache;
    this.logger = opts.logger;
    this.db = opts.db;
    this.metrics = opts.metrics;
    this.rateLimiter = opts.rateLimiter;
    this.usageLimiter = opts.usageLimiter;
    this.rbac = opts.rbac;
  }

  public async verifyKey(
    c: Context,
    req: {
      key: string;
      apiId?: string;
      permissionQuery?: PermissionQuery;
      ratelimit?: { cost?: number };
      ratelimits?: Array<Omit<RatelimitRequest, "identity">>;
    },
  ): Promise<
    Result<
      VerifyKeyResult,
      SchemaError | FetchError | DisabledWorkspaceError | MissingRatelimitError
    >
  > {
    try {
      const res = await this._verifyKey(c, req);
      if (res.err) {
        this.metrics.emit({
          metric: "metric.key.verification",
          valid: false,
          code: res.err.message,
        });
        return res;
      }
      c.set("workspaceId", res.val.key?.forWorkspaceId ?? res.val.key?.workspaceId);

      this.metrics.emit({
        metric: "metric.key.verification",
        valid: res.val.valid,
        code: res.val.code ?? "OK",
        workspaceId: res.val.key?.workspaceId,
        apiId: res.val.api?.id,
        keyId: res.val.key?.id,
      });

      return res;
    } catch (e) {
      const err = e as Error;
      this.logger.error("Unhandled error while verifying key", {
        error: err.message,
        stack: JSON.stringify(err.stack),
        keyHash: await this.hash(req.key),
        apiId: req.apiId,
      });

      throw e;
    }
  }

  private async hash(key: string): Promise<string> {
    const cached = this.hashCache.get(key);
    if (cached) {
      return cached;
    }
    const hash = await sha256(key);
    this.hashCache.set(key, hash);
    return hash;
  }
  /**
   * extracting this into a separate function just makes it easier to emit the analytics event
   */
  private async _verifyKey(
    c: Context,
    req: {
      key: string;
      apiId?: string;
      permissionQuery?: PermissionQuery;
      ratelimit?: { cost?: number };
      ratelimits?: Array<Omit<RatelimitRequest, "identity">>;
    },
  ): Promise<
    Result<
      VerifyKeyResult,
      FetchError | SchemaError | DisabledWorkspaceError | MissingRatelimitError
    >
  > {
    const keyHash = await this.hash(req.key);
    const { val: data, err } = await retry(
      3,
      async () =>
        this.cache.keyByHash.swr(keyHash, async (hash) => {
          const dbStart = performance.now();
          const dbRes = await this.db.readonly.query.keys.findFirst({
            where: (table, { and, eq, isNull }) =>
              and(eq(table.hash, hash), isNull(table.deletedAt)),
            with: {
              encrypted: true,
              workspace: {
                columns: {
                  id: true,
                  enabled: true,
                },
              },
              forWorkspace: {
                columns: {
                  id: true,
                  enabled: true,
                },
              },
              roles: {
                with: {
                  role: {
                    with: {
                      permissions: {
                        with: {
                          permission: true,
                        },
                      },
                    },
                  },
                },
              },
              permissions: {
                with: {
                  permission: true,
                },
              },
              keyAuth: {
                with: {
                  api: true,
                },
              },
              ratelimits: true,
              identity: {
                with: {
                  ratelimits: true,
                },
              },
            },
          });
          this.metrics.emit({
            metric: "metric.db.read",
            query: "getKeyAndApiByHash",
            latency: performance.now() - dbStart,
          });
          if (!dbRes?.keyAuth?.api) {
            return null;
          }
          if (dbRes.keyAuth.deletedAt) {
            return null;
          }
          if (dbRes.keyAuth.api.deletedAt) {
            return null;
          }

          /**
           * Createa a unique set of all permissions, whether they're attached directly or connected
           * through a role.
           */
          const permissions = new Set<string>([
            ...dbRes.permissions.map((p) => p.permission.name),
            ...dbRes.roles.flatMap((r) => r.role.permissions.map((p) => p.permission.name)),
          ]);

          /**
           * Merge ratelimits from the identity and the key
           * Key limits take pecedence
           */
          const ratelimits: { [name: string]: Pick<Ratelimit, "name" | "limit" | "duration"> } = {};

          if (
            dbRes.ratelimitAsync !== null &&
            dbRes.ratelimitDuration !== null &&
            dbRes.ratelimitLimit !== null
          ) {
            ratelimits.default = {
              name: "default",
              limit: dbRes.ratelimitLimit,
              duration: dbRes.ratelimitDuration,
            };
          }
          for (const rl of dbRes.identity?.ratelimits ?? []) {
            ratelimits[rl.name] = rl;
          }
          for (const rl of dbRes.ratelimits ?? []) {
            ratelimits[rl.name] = rl;
          }

          return {
            workspace: dbRes.workspace,
            forWorkspace: dbRes.forWorkspace,
            key: dbRes,
            identity: dbRes.identity,
            api: dbRes.keyAuth.api,
            permissions: Array.from(permissions.values()),
            roles: dbRes.roles.map((r) => r.role.name),
            ratelimits,
          };
        }),
      (attempt, err) => {
        this.logger.warn("Failed to fetch key data, retrying...", {
          hash: keyHash,
          attempt,
          error: err.message,
        });
      },
    );

    if (err) {
      this.logger.error(err.message, {
        hash: keyHash,
        error: err,
      });

      return Err(
        new FetchError({
          message: "unable to fetch required data",
          retry: true,
          cause: err,
        }),
      );
    }

    if (!data) {
      return Ok({ valid: false, code: "NOT_FOUND" });
    }

    if ((data.forWorkspace && !data.forWorkspace.enabled) || !data.workspace.enabled) {
      return Err(new DisabledWorkspaceError(data.workspace.id));
    }

    /**
     * Enabled
     */
    if (data.key.enabled === false) {
      return Ok({
        key: data.key,
        identity: data.identity,
        api: data.api,
        valid: false,
        code: "DISABLED",
        permissions: data.permissions,
        message: "the key is disabled",
      });
    }

    if (req.apiId && data.api?.id !== req.apiId) {
      return Ok({
        key: data.key,
        api: data.api,
        identity: data.identity,
        valid: false,
        code: "FORBIDDEN",
        permissions: data.permissions,
        message: `the key does not belong to ${req.apiId}`,
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
        return Ok({
          valid: false,
          code: "EXPIRED",
          key: data.key,
          api: data.api,
          identity: data.identity,
          permissions: data.permissions,
          message: `the key has expired on ${new Date(expires).toISOString()}`,
        });
      }
    }

    if (data.api.ipWhitelist) {
      const ip = c.req.header("True-Client-IP") ?? c.req.header("CF-Connecting-IP");
      if (!ip) {
        return Ok({
          key: data.key,
          identity: data.identity,
          api: data.api,
          valid: false,
          code: "FORBIDDEN",
          permissions: data.permissions,
        });
      }
      const ipWhitelist = JSON.parse(data.api.ipWhitelist) as string[];
      if (!ipWhitelist.includes(ip)) {
        return Ok({
          key: data.key,
          identity: data.identity,
          api: data.api,
          valid: false,
          code: "FORBIDDEN",
          permissions: data.permissions,
        });
      }
    }

    if (req.permissionQuery) {
      const q = this.rbac.validateQuery(req.permissionQuery);
      if (q.err) {
        return Err(
          new SchemaError({
            message: "permission query is invalid",
            cause: q.err,
            context: {
              raw: req.permissionQuery,
            },
          }),
        );
      }
      const rbacResp = this.rbac.evaluatePermissions(q.val.query, data.permissions);

      if (rbacResp.err) {
        this.logger.error("evaluating permissions failed", {
          query: JSON.stringify(req.permissionQuery),
          permissions: JSON.stringify(data.permissions),
        });
        return Err(
          new SchemaError({
            message: "permission query is invalid",
            cause: q.err,
            context: {
              raw: req.permissionQuery,
            },
          }),
        );
      }
      if (!rbacResp.val.valid) {
        return Ok({
          key: data.key,
          identity: data.identity,
          api: data.api,
          valid: false,
          code: "INSUFFICIENT_PERMISSIONS",
          permissions: data.permissions,
          message: rbacResp.val.message,
        });
      }
    }

    /**
     * Ratelimiting
     */

    const ratelimits: {
      [name: string | "default"]: Required<RatelimitRequest>;
    } = {};
    if ("default" in data.ratelimits && typeof req.ratelimits === "undefined") {
      ratelimits.default = {
        identity: data.key.id,
        name: data.ratelimits.default.name,
        cost: req.ratelimit?.cost ?? 1,
        limit: data.ratelimits.default.limit,
        duration: data.ratelimits.default.duration,
      };
    }

    for (const r of req.ratelimits ?? []) {
      if (typeof r.limit !== "undefined" && typeof r.duration !== "undefined") {
        ratelimits[r.name] = {
          identity: data.identity?.id ?? data.key.id,
          name: r.name,
          cost: r.cost ?? 1,
          limit: r.limit,
          duration: r.duration,
        };
        continue;
      }

      const configured = data.ratelimits[r.name];
      if (configured) {
        ratelimits[configured.name] = {
          identity: data.identity?.id ?? data.key.id,
          name: configured.name,
          cost: r.cost ?? 1,
          limit: configured.limit,
          duration: configured.duration,
        };
        continue;
      }

      let errorMessage = `ratelimit "${r.name}" was requested but does not exist for key "${data.key.id}"`;
      if (data.identity) {
        errorMessage += ` nor identity { id: ${data.identity.id}, externalId: ${data.identity.externalId}}`;
      } else {
        errorMessage += " and there is no identity connected";
      }

      return Err(new MissingRatelimitError(r.name, errorMessage));
    }

    const [pass, ratelimit] = await this.ratelimit(c, data.key, ratelimits);

    if (!pass) {
      return Ok({
        key: data.key,
        api: data.api,
        identity: data.identity,
        valid: false,
        code: "RATE_LIMITED",
        ratelimit,
        permissions: data.permissions,
      });
    }

    let remaining: number | undefined = undefined;
    if (data.key.remaining !== null) {
      const limited = await this.usageLimiter.limit({ keyId: data.key.id });
      remaining = limited.remaining;
      if (!limited.valid) {
        return Ok({
          key: data.key,
          api: data.api,
          identity: data.identity,
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
          permissions: data.permissions,
        });
      }
    }

    return Ok({
      workspaceId: data.key.workspaceId,
      key: data.key,
      identity: data.identity,
      api: data.api,
      valid: true,
      ownerId: data.key.ownerId ?? undefined,
      expires,
      ratelimit,
      remaining,
      isRootKey: !!data.key.forWorkspaceId,
      authorizedWorkspaceId: data.key.forWorkspaceId ?? data.key.workspaceId,
      permissions: data.permissions,
    });
  }

  /**
   * @returns [pass, ratelimit]
   */
  private async ratelimit(
    c: Context,
    key: Key,
    ratelimits: { [name: string | "default"]: Required<RatelimitRequest> },
  ): Promise<[boolean, VerifyKeyResult["ratelimit"]]> {
    if (Object.keys(ratelimits).length === 0) {
      return [true, undefined];
    }
    if (!this.rateLimiter) {
      this.logger.error("ratelimiting is not enabled, but a key has ratelimiting enabled");
      return [true, undefined];
    }

    const res = await this.rateLimiter.multiLimit(
      c,
      Object.values(ratelimits).map((r) => ({
        name: r.name,
        async: !!key.ratelimitAsync,
        workspaceId: key.workspaceId,
        identifier: r.identity,
        cost: r.cost,
        interval: r.duration,
        limit: r.limit,
      })),
    );

    if (res.err) {
      this.logger.error("ratelimiting failed", {
        error: res.err.message,
        ...res.err,
      });

      return [false, undefined];
    }

    if (res.val.triggered !== null) {
      /**
       * This is undocumented and only used internally for test assertions
       */
      c.res.headers.set("Unkey-Ratelimit-Triggered", res.val.triggered);
    }

    return [
      res.val.pass,
      {
        remaining: res.val.remaining,
        limit: ratelimits.default?.limit,
        reset: res.val.reset,
      },
    ];
  }
}
