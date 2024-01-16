/**
 * The database takes care of isolating roles between workspaces.
 * That's why we can assume the highest scope of a role is an `api` or later `gateway`
 *
 * role identifiers can look like this:
 * - `api_id.xxx`
 * - `gateway_id.xxx`
 *
 */

import { z } from "zod";
import { Flatten } from "./types";

export function buildIdSchema(prefix: string) {
  return z.string().refine((s) => {
    if (s === "*") {
      return true;
    }
    const regex = new RegExp(
      `^${prefix}_[123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz]{8,32}$`,
    );
    return regex.test(s);
  });
}
const apiId = buildIdSchema("api");
const rootKeyId = buildIdSchema("unkey");

export const rootKeyActions = z.enum([
  "read_root_key",
  "create_root_key",
  "delete_root_key",
  "update_root_key",
]);

export const apiActions = z.enum([
  "read_api",
  "create_api",
  "delete_api",
  "update_api",
  "create_key",
  "update_key",
  "delete_key",
  "read_key",
]);

export type Resources = {
  [resourceId in `api.${z.infer<typeof apiId>}`]: z.infer<typeof apiActions>;
} & {
  [resourceId in `root_key.${z.infer<typeof rootKeyId>}`]: z.infer<typeof rootKeyActions>;
};

export type Role = Flatten<Resources>;

/**
 * Validation for roles used for our root keys
 */
export const unkeyRoleValidation = z.string().refine((s) => {
  const split = s.split(".");
  if (split.length !== 3) {
    return false;
  }
  const [resource, id, action] = split;
  switch (resource) {
    case "api": {
      return apiId.safeParse(id).success && apiActions.safeParse(action).success;
    }
    case "root_key": {
      return rootKeyId.safeParse(id).success && rootKeyActions.safeParse(action).success;
    }
    default: {
      return false;
    }
  }
});
