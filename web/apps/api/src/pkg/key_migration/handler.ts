import { BaseError, Err, Ok, type Result } from "@unkey/error";

import { newId } from "@unkey/id";
import { ConsoleLogger } from "@unkey/worker-logging";
import { createConnection, schema } from "../db";
import type { Env } from "../env";
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
    retry: 3,
    logger: new ConsoleLogger({
      requestId: "",
      application: "api",
      environment: env.ENVIRONMENT,
    }),
  });

  const keyId = newId("key");

  // name -> id
  const roles: Record<string, string> = {};

  if (message.roles && message.roles.length > 0) {
    const found = await db.query.roles.findMany({
      where: (table, { inArray, and, eq }) =>
        and(eq(table.workspaceId, message.workspaceId), inArray(table.name, message.roles ?? [])),
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
          eq(table.workspaceId, message.workspaceId),
          inArray(table.name, message.permissions ?? []),
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
  try {
    await db.transaction(async (tx) => {
      await tx.insert(schema.keys).values({
        id: keyId,
        workspaceId: message.workspaceId,
        keyAuthId: message.keyAuthId,
        hash: message.hash,
        start: message.start ?? "",
        ownerId: message.ownerId,
        meta: message.meta ? JSON.stringify(message.meta) : null,
        createdAtM: Date.now(),
        expires: message.expires ? new Date(message.expires) : null,
        refillAmount: message.refill?.amount,
        refillDay: message.refill?.refillDay,
        enabled: message.enabled,
        remaining: message.remaining,
        environment: message.environment,
      });
      if (message.ratelimit) {
        await tx.insert(schema.ratelimits).values({
          id: newId("ratelimit"),
          workspaceId: message.workspaceId,
          keyId: keyId,
          limit: message.ratelimit.limit,
          duration: message.ratelimit.duration,
          name: "default",
        });
      }

      if (message.encrypted) {
        await tx.insert(schema.encryptedKeys).values({
          workspaceId: message.workspaceId,
          keyId: keyId,
          encrypted: message.encrypted.encrypted,
          encryptionKeyId: message.encrypted.keyId,
        });
      }

      /**
       * ROLES
       */

      if (Object.keys(roles).length > 0) {
        const roleConnections = Object.values(roles).map((roleId) => ({
          keyId,
          roleId,
          workspaceId: message.workspaceId,
          createdAtM: Date.now(),
        }));

        await tx.insert(schema.keysRoles).values(roleConnections);
      }

      /**
       * PERMISSIONS
       */

      if (Object.keys(permissions).length > 0) {
        const permissionConnections = Object.values(permissions).map((permissionId) => ({
          keyId,
          permissionId,
          workspaceId: message.workspaceId,
          createdAtM: Date.now(),
        }));

        await tx.insert(schema.keysPermissions).values(permissionConnections);
      }
    });
  } catch (e) {
    const err = e as Error;
    return Err(
      new MigrationError({
        message: err.message,
      }),
    );
  }
  return Ok();
}
