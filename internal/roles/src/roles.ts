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

export const apiId = z.string().regex(/^api_[123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz]{8,32}$/).or(z.literal("*"))
export const keyId = z.string().regex(/^key_[123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz]{8,32}$/).or(z.literal("*"))


const wildcard = z.literal("*")
const delimiter = z.literal("::")

const resourceId = z.union([apiId, keyId, wildcard])

const apiActions = z.enum(["create", "read", "update", "delete", "createKey", "updateKey", "deleteKey"])
const keyActions = z.enum(["read", "update", "delete"])


const roles = z.union([
  z.tuple([apiId, apiActions]),
  z.tuple([keyId, keyActions]),
])
