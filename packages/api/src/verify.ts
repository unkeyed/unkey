import type { PermissionQuery } from "@unkey/rbac";
import { Unkey } from "./client";
import type { paths } from "./openapi";

/**
 * Verify a key
 *
 * @example
 * ```ts
 * const { result, error } = await verifyKey("key_123")
 * if (error){
 *   // handle potential network or bad request error
 *   // a link to our docs will be in the `error.docs` field
 *   console.error(error.message)
 *   return
 * }
 * if (!result.valid) {
 *   // do not grant access
 *   return
 * }
 *
 * // process request
 * console.log(result)
 * ```
 */
export function verifyKey<TPermission extends string = string>(
  req:
    | string
    | (Omit<
        paths["/v1/keys.verifyKey"]["post"]["requestBody"]["content"]["application/json"],
        "authorization"
      > & { authorization?: { permissions: PermissionQuery<TPermission> } }),
) {
  // yes this is empty to make typescript happy but we don't need a token for verifying keys
  // it's not the cleanest but it works for now :)
  const unkey = new Unkey({ rootKey: "public" });
  return unkey.keys.verify(typeof req === "string" ? { key: req } : req);
}
