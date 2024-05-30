import { BaseError, Err, Ok, type Result } from "@unkey/error";

import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { Analytics } from "../analytics";
import { createConnection, schema } from "../db";
import type { Env } from "../env";
import { connectVault } from "../vault";
import type { MessageBody } from "./message";

export class MigrationError extends BaseError {
  readonly name = "MigrationError";
  readonly retry = false;
}

export async function migrateKey(
  message: MessageBody,
  env: Env,
): Promise<Result<{ keyId: string }, MigrationError>> {
  const db = createConnection({
    host: env.DATABASE_HOST,
    username: env.DATABASE_USERNAME,
    password: env.DATABASE_PASSWORD,
  });
  const vault = connectVault(env);

  const tinybirdProxy =
    env.TINYBIRD_PROXY_URL && env.TINYBIRD_PROXY_TOKEN
      ? {
          url: env.TINYBIRD_PROXY_URL,
          token: env.TINYBIRD_PROXY_TOKEN,
        }
      : undefined;

  const analytics = new Analytics({
    tinybirdProxy,
    tinybirdToken: env.TINYBIRD_TOKEN,
  });

  const migration = await db.query.keyMigrations.findFirst({
    where: (table, { eq }) => eq(table.id, message.migrationId),
  });
  if (!migration) {
    return Err(
      new MigrationError({
        message: "Migration not found",
      }),
    );
  }

  const keyId = newId("key");
  const hash = message.plaintext ? await sha256(message.plaintext) : message.hash;
  if (!hash) {
    return Err(
      new MigrationError({
        message: "Hash is required",
      }),
    );
  }

  const encrypted = message.plaintext
    ? await vault.encrypt({
        keyring: migration.workspaceId,
        data: message.plaintext,
      })
    : null;

  // name -> id
  const roles: Record<string, string> = {};

  if (message.roles && message.roles.length > 0) {
    const found = await db.query.roles.findMany({
      where: (table, { inArray, and, eq }) =>
        and(eq(table.workspaceId, migration.workspaceId), inArray(table.name, message.roles!)),
    });
    const missingRoles = message.roles.filter((name) => !found.some((role) => role.name === name));
    if (missingRoles.length > 0) {
      return Err(
        new MigrationError({
          message: `Roles ${JSON.stringify(missingRoles)} are missing, please create them first`,
        }),
      );
    }
    for (const role of found) {
      roles[role.name] = role.id;
    }
  }
  // name -> id
  const permissions: Record<string, string> = {};

  if (message.permissions && message.permissions.length > 0) {
    const found = await db.query.permissions.findMany({
      where: (table, { inArray, and, eq }) =>
        and(
          eq(table.workspaceId, migration.workspaceId),
          inArray(table.name, message.permissions!),
        ),
    });
    const missingRoles = message.permissions.filter(
      (name) => !found.some((permission) => permission.name === name),
    );
    if (missingRoles.length > 0) {
      return Err(
        new MigrationError({
          message: `Roles ${JSON.stringify(missingRoles)} are missing, please create them first`,
        }),
      );
    }
    for (const permission of found) {
      permissions[permission.name] = permission.id;
    }
  }

  await db.transaction(async (tx) => {
    await tx.insert(schema.keys).values({
      id: keyId,
      workspaceId: migration.workspaceId,
      keyAuthId: migration.keyAuthId,
      hash,
      start: message.start ?? message.plaintext?.slice(0, 4) ?? "",
      ownerId: message.ownerId,
      meta: message.meta ? JSON.stringify(message.meta) : null,
      createdAt: new Date(),
      createdAtM: Date.now(),
      expires: message.expires ? new Date(message.expires) : null,
      refillInterval: message.refill?.interval,
      refillAmount: message.refill?.amount,
      enabled: message.enabled,
      remaining: message.remaining,
      ratelimitAsync: message.ratelimit?.async,
      ratelimitLimit: message.ratelimit?.limit,
      ratelimitDuration: message.ratelimit?.duration,
      environment: message.environment,
    });

    await analytics.ingestUnkeyAuditLogs({
      workspaceId: migration.workspaceId,
      event: "key.create",
      actor: {
        type: "key",
        id: message.rootKeyId,
      },
      description: `Created ${keyId} in ${migration.keyAuthId}`,
      resources: [
        {
          type: "key",
          id: keyId,
        },
        {
          type: "keyAuth",
          id: migration.keyAuthId!,
        },
      ],

      context: message.auditLogContext,
    });

    if (encrypted) {
      await tx.insert(schema.encryptedKeys).values({
        workspaceId: migration.workspaceId,
        keyId: keyId,
        encrypted: encrypted.encrypted,
        encryptionKeyId: encrypted.keyId,
      });
    }

    /**
     * ROLES
     */

    if (Object.keys(roles).length > 0) {
      const roleConnections = Object.values(roles).map((roleId) => ({
        keyId,
        roleId,
        workspaceId: migration.workspaceId,
        createdAt: new Date(),
      }));

      await tx.insert(schema.keysRoles).values(roleConnections);

      await analytics.ingestUnkeyAuditLogs(
        roleConnections.map((rc) => ({
          workspaceId: migration.workspaceId,
          actor: { type: "key", id: message.rootKeyId },
          event: "authorization.connect_role_and_key",
          description: `Connected ${rc.roleId} and ${rc.keyId}`,
          resources: [
            {
              type: "key",
              id: rc.keyId,
            },
            {
              type: "role",
              id: rc.roleId,
            },
          ],
          context: message.auditLogContext,
        })),
      );
    }

    /**
     * PERMISSIONS
     */

    if (Object.keys(permissions).length > 0) {
      const permissionConnections = Object.values(permissions).map((permissionId) => ({
        keyId,
        permissionId,
        workspaceId: migration.workspaceId,
        createdAt: new Date(),
      }));

      await tx.insert(schema.keysPermissions).values(permissionConnections);

      await analytics.ingestUnkeyAuditLogs(
        permissionConnections.map((pc) => ({
          workspaceId: migration.workspaceId,
          actor: { type: "key", id: message.rootKeyId },
          event: "authorization.connect_permission_and_key",
          description: `Connected ${pc.permissionId} and ${pc.keyId}`,
          resources: [
            {
              type: "key",
              id: pc.keyId,
            },
            {
              type: "permission",
              id: pc.permissionId,
            },
          ],
          context: message.auditLogContext,
        })),
      );
    }
  });

  return Ok();
}
