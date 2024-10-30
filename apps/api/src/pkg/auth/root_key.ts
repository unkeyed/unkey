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

  const { keyService, analytics } = c.get("services");
  const { val: rootKey, err } = await keyService.verifyKey(c, {
    key: authorization,
    permissionQuery,
  });

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

  if (!rootKey.key) {
    throw new UnkeyApiError({
      code: "UNAUTHORIZED",
      message: "key not found",
    });
  }

  // if we have identified the key, we can send the analytics event
  // otherwise, they likely sent garbage to us and we can't associate it with anything

  c.executionCtx.waitUntil(
    analytics.ingestKeyVerification({
      workspaceId: rootKey.key.workspaceId,
      apiId: rootKey.api.id,
      keyId: rootKey.key.id,
      time: Date.now(),
      deniedReason: rootKey.code,
      ipAddress: c.req.header("True-Client-IP") ?? c.req.header("CF-Connecting-IP"),
      userAgent: c.req.header("User-Agent"),
      requestedResource: "",
      // @ts-expect-error - the cf object will be there on cloudflare
      region: c.req.raw?.cf?.country ?? "",
      ownerId: rootKey.key.ownerId ?? undefined,
      // @ts-expect-error - the cf object will be there on cloudflare
      edgeRegion: c.req.raw?.cf?.colo ?? "",
      keySpaceId: rootKey.key.keyAuthId,
      requestId: c.get("requestId"),
    }),
  );

  if (!rootKey.valid) {
    throw new UnkeyApiError({
      code: rootKey.code,
      message: "message" in rootKey && rootKey.message ? rootKey.message : "unauthorized",
    });
  }
  if (!rootKey.isRootKey) {
    throw new UnkeyApiError({
      code: "UNAUTHORIZED",
      message: "root key required",
    });
  }

  return rootKey;
}
