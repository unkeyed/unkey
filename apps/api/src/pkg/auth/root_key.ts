import { SchemaError } from "@unkey/error";
import type { PermissionQuery } from "@unkey/rbac";
import type { Context } from "hono";
import { UnkeyApiError } from "../errors";
import type { HonoEnv } from "../hono/env";
import { DisabledWorkspaceError } from "../keys/service";

/**
 * rootKeyAuth takes the bearer token from the request and verifies the key
 *
 * if the key doesnt exist, isn't valid or isn't a root key, an error is thrown, which gets handled
 * automatically by hono
 */
export async function rootKeyAuth(c: Context<HonoEnv>, permissionQuery?: PermissionQuery) {
  const authorization = c.req.header("authorization")?.replace("Bearer ", "");
  if (!authorization) {
    throw new UnkeyApiError({ code: "UNAUTHORIZED", message: "key required" });
  }

  const svc = c.get("services").keyService;
  const { val: rootKey, err } = await svc.verifyKey(c, { key: authorization, permissionQuery });

  if (err) {
    switch (true) {
      case err instanceof SchemaError:
        throw new UnkeyApiError({
          code: "BAD_REQUEST",
          message: err.message,
        });
      case err instanceof DisabledWorkspaceError:
        throw new UnkeyApiError({
          code: "FORBIDDEN",
          message: "workspace is disabled",
        });
    }
    throw new UnkeyApiError({
      code: "INTERNAL_SERVER_ERROR",
      message: err.message,
    });
  }
  if (!rootKey.valid) {
    throw new UnkeyApiError({
      code: rootKey.code === "NOT_FOUND" ? "UNAUTHORIZED" : rootKey.code,
      message: "unauthorized",
    });
  }
  if (!rootKey.isRootKey) {
    throw new UnkeyApiError({ code: "UNAUTHORIZED", message: "root key required" });
  }

  return rootKey;
}
