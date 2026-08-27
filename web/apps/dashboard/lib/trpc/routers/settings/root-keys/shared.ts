import { eq, gt, isNull, or, schema } from "@/lib/db";

export type RootKeyPermission = {
  id: string;
  name: string;
};

export function rootKeyBaseConditions(workspaceId: string) {
  return [
    eq(schema.keys.forWorkspaceId, workspaceId),
    isNull(schema.keys.deletedAtM),
    or(isNull(schema.keys.expires), gt(schema.keys.expires, new Date())),
  ];
}

export function rootKeyPermissions(
  rows: readonly { permission: RootKeyPermission }[],
): RootKeyPermission[] {
  return rows.map((row) => ({ id: row.permission.id, name: row.permission.name }));
}

export function rootKeyGrants(rows: readonly { permission: { name: string } }[]): string[] {
  return [...new Set(rows.map((row) => row.permission.name))];
}
