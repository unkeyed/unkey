/**
 * The database takes care of isolating roles between workspaces.
 * That's why we can assume the highest scope of a role is an `api` or later `gateway`
 *
 * role identifiers can look like this:
 * - `api_id::xxx`
 * - `gateway_id::xxx`
 *
 */

import { z } from "zod";
import { Flatten } from "./types";

export function buildIdSchema<TPrefix extends string>(prefix: TPrefix) {
  return z.custom<`${TPrefix}_${string}` | `*`>((s) => {
    if (typeof s !== "string") {
      return false;
    }
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
const keyId = buildIdSchema("key");

const rootKeyActions = z.enum([
  "read_root_key",
  "create_root_key",
  "delete_root_key",
  "update_root_key",
]);
const apiActions = z.enum([
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
  [resourceId in `api::${z.infer<typeof apiId>}`]: z.infer<typeof apiActions>;
} & {
  [resourceId in `root_key::${z.infer<typeof keyId>}`]: z.infer<typeof rootKeyActions>;
};

export type Roles = Flatten<Resources, "::">;

// const queries = {
//   and: (...args: unknown[]) => {
//     return {
//       op: "and",
//       set: args,
//     };
//   },
//   or: (...args: unknown[]) => {
//     return {
//       op: "or",
//       set: args,
//     };
//   },
// };

// root_key::*::read_root_key
// root_key::*::create_root_key // a root key MUST NOT be allowed to create another key with more permissions than itself
// root_key::*::delete_root_key
// root_key::*::update_root_key
// api::*::create_api
// api::*::delete_api // either wildcard or a specific id -> api::api_123::delete_api
// api::*::read_api
// api::*::update_api
// api::*::read_key
// api::*::create_key
// api::*::update_key
// api::*::delete_key
