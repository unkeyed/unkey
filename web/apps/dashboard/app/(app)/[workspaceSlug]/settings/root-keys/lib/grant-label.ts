import { parseUrnPermissionParts } from "@unkey/rbac";
import { describePermission, humaniseAction } from "./describe-permission";

export type GrantLabel = {
  path: string | null;
  action: string;
};

export function grantLabel(grant: string): GrantLabel {
  const parsed = parseUrnPermissionParts(grant);
  if (parsed === null) {
    return { path: null, action: describePermission(grant) };
  }
  return { path: parsed.resourcePath, action: humaniseAction(parsed.action) };
}
