import { initTRPC } from "@trpc/server";
import { newId } from "@unkey/id";
import { beforeEach, expect, it, vi } from "vitest";
import { z } from "zod";
import { updateRole } from "./update";

type Context = {
  workspace: { id: string };
  user: { id: string };
  audit: { location: string; userAgent: string };
};

const state = vi.hoisted(() => ({
  roleId: "",
  permissionIds: [] as string[],
  keyIds: [] as string[],
  availablePermissionIds: [] as string[],
  availableKeyIds: [] as string[],
  outbox: [] as unknown[],
}));

vi.mock("@/lib/trpc/trpc", async () => {
  const { initTRPC } = await import("@trpc/server");
  return { workspaceProcedure: initTRPC.context<Context>().create().procedure };
});

vi.mock("@/lib/db", async () => {
  const actual = await vi.importActual<typeof import("@unkey/db")>("@unkey/db");
  const tx = {
    query: {
      roles: { findFirst: async () => ({ id: state.roleId, name: "Editor" }) },
      rolesPermissions: {
        findMany: async () => state.permissionIds.map((permissionId) => ({ permissionId })),
      },
      keysRoles: { findMany: async () => state.keyIds.map((keyId) => ({ keyId })) },
      permissions: { findMany: async () => state.availablePermissionIds.map((id) => ({ id })) },
      keys: { findMany: async () => state.availableKeyIds.map((id) => ({ id })) },
    },
    update: () => ({ set: () => ({ where: async () => undefined }) }),
    delete: (table: unknown) => ({
      where: async () => {
        if (table === actual.schema.rolesPermissions) {
          state.permissionIds = [];
        } else if (table === actual.schema.keysRoles) {
          state.keyIds = [];
        } else {
          throw new Error("Unexpected table deletion");
        }
      },
    }),
    insert: (table: unknown) => ({
      values: async (rows: unknown) => {
        if (table === actual.schema.rolesPermissions) {
          state.permissionIds = z
            .array(z.object({ permissionId: z.string() }))
            .parse(rows)
            .map((row) => row.permissionId);
        } else if (table === actual.schema.keysRoles) {
          state.keyIds = z
            .array(z.object({ keyId: z.string() }))
            .parse(rows)
            .map((row) => row.keyId);
        } else if (table === actual.schema.clickhouseOutbox) {
          state.outbox.push(...z.array(z.unknown()).parse(rows));
        } else {
          throw new Error("Unexpected table insertion");
        }
      },
    }),
  };
  return {
    ...actual,
    db: { transaction: async (run: (value: typeof tx) => Promise<void>) => run(tx) },
  };
});

const ctx: Context = {
  workspace: { id: newId("workspace") },
  user: { id: newId("test") },
  audit: { location: "127.0.0.1", userAgent: "vitest" },
};
const caller = initTRPC.context<Context>().create().router({ updateRole }).createCaller(ctx);
const eventSchema = z.object({
  event: z.string(),
  correlation_id: z.string(),
  actor: z.object({ type: z.string(), id: z.string() }),
  targets: z.array(z.object({ type: z.string(), id: z.string() })),
});

beforeEach(() => {
  state.roleId = newId("role");
  state.permissionIds = [newId("permission"), newId("permission"), newId("permission")];
  state.keyIds = [newId("key"), newId("key")];
  state.availablePermissionIds = [...state.permissionIds];
  state.availableKeyIds = [...state.keyIds];
  state.outbox = [];
});

it("audits removed endpoints while preserving connection events for replacement bindings", async () => {
  const [removedPermission, retainedPermission, otherRemovedPermission] = state.permissionIds;
  const [removedKey, retainedKey] = state.keyIds;
  if (
    !removedPermission ||
    !retainedPermission ||
    !otherRemovedPermission ||
    !removedKey ||
    !retainedKey
  ) {
    throw new Error("Missing fixture endpoints");
  }
  const addedPermission = newId("permission");
  const addedKey = newId("key");
  state.availablePermissionIds.push(addedPermission);
  state.availableKeyIds.push(addedKey);
  await caller.updateRole({
    roleId: state.roleId,
    roleName: "Editor",
    permissionIds: [retainedPermission, addedPermission],
    keyIds: [retainedKey, addedKey],
  });
  const events = state.outbox.map((row) => z.object({ payload: eventSchema }).parse(row).payload);
  const expected = [
    [
      "authorization.disconnect_role_and_permissions",
      [
        { type: "role", id: state.roleId },
        { type: "permission", id: removedPermission },
      ],
    ],
    [
      "authorization.disconnect_role_and_permissions",
      [
        { type: "role", id: state.roleId },
        { type: "permission", id: otherRemovedPermission },
      ],
    ],
    [
      "authorization.connect_role_and_permission",
      [
        { type: "role", id: state.roleId },
        { type: "permission", id: retainedPermission },
      ],
    ],
    [
      "authorization.disconnect_role_and_key",
      [
        { type: "key", id: removedKey },
        { type: "role", id: state.roleId },
      ],
    ],
    [
      "authorization.connect_role_and_permission",
      [
        { type: "role", id: state.roleId },
        { type: "permission", id: addedPermission },
      ],
    ],
    [
      "authorization.connect_role_and_key",
      [
        { type: "key", id: retainedKey },
        { type: "role", id: state.roleId },
      ],
    ],
    [
      "authorization.connect_role_and_key",
      [
        { type: "key", id: addedKey },
        { type: "role", id: state.roleId },
      ],
    ],
    ["role.update", [{ type: "role", id: state.roleId }]],
  ];
  expect(events.map((event) => [event.event, event.targets])).toEqual(
    expect.arrayContaining(expected),
  );
  expect(events).toHaveLength(expected.length);
  expect(new Set(events.map((event) => event.correlation_id)).size).toBe(1);
  expect(
    events.every((event) => event.actor.type === "user" && event.actor.id === ctx.user.id),
  ).toBe(true);
  expect(state.permissionIds).toEqual([retainedPermission, addedPermission]);
  expect(state.keyIds).toEqual([retainedKey, addedKey]);
});

it.each([
  { permissions: true, keys: true },
  { permissions: true, keys: false },
  { permissions: false, keys: true },
  { permissions: false, keys: false },
])("audits [] clears but leaves omitted bindings unchanged: %j", async ({ permissions, keys }) => {
  const permissionIds = [...state.permissionIds];
  const keyIds = [...state.keyIds];
  await caller.updateRole({
    roleId: state.roleId,
    roleName: "Editor",
    ...(permissions ? { permissionIds: [] } : {}),
    ...(keys ? { keyIds: [] } : {}),
  });
  const events = state.outbox.map((row) => z.object({ payload: eventSchema }).parse(row).payload);
  const expected = [
    ...(permissions
      ? permissionIds.map((id) => ({
          event: "authorization.disconnect_role_and_permissions",
          targets: [
            { type: "role", id: state.roleId },
            { type: "permission", id },
          ],
        }))
      : []),
    ...(keys
      ? keyIds.map((id) => ({
          event: "authorization.disconnect_role_and_key",
          targets: [
            { type: "key", id },
            { type: "role", id: state.roleId },
          ],
        }))
      : []),
    { event: "role.update", targets: [{ type: "role", id: state.roleId }] },
  ];
  expect(events.map(({ event, targets }) => ({ event, targets }))).toEqual(
    expect.arrayContaining(expected),
  );
  expect(events).toHaveLength(expected.length);
  expect(new Set(events.map((event) => event.correlation_id)).size).toBe(1);
  expect(state.permissionIds).toEqual(permissions ? [] : permissionIds);
  expect(state.keyIds).toEqual(keys ? [] : keyIds);
});
