import { db, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { newKey } from "@unkey/keys";
import { z } from "zod";
import { auth, t } from "../trpc";

export const keyRouter = t.router({
  create: t.procedure
    .use(auth)
    .input(
      z.object({
        prefix: z.string().optional(),
        bytesLength: z.number().int().gte(1).default(16),
        apiId: z.string(),
        ownerId: z.string().nullish(),
        meta: z.record(z.unknown()).optional(),
        remaining: z.number().int().positive().optional(),
        expires: z.number().int().nullish(), // unix timestamp in milliseconds
        name: z.string().optional(),
        ratelimit: z
          .object({
            type: z.enum(["consistent", "fast"]),
            refillInterval: z.number().int().positive(),
            refillRate: z.number().int().positive(),
            limit: z.number().int().positive(),
          })
          .optional(),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      const workspace = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      });
      if (!workspace) {
        throw new TRPCError({ code: "NOT_FOUND", message: "workspace not found" });
      }

      const api = await db.query.apis.findFirst({
        where: (table, { eq }) => eq(table.id, input.apiId),
      });
      if (!api) {
        throw new TRPCError({ code: "NOT_FOUND", message: "api not found" });
      }
      if (!api.keyAuthId) {
        throw new TRPCError({
          code: "PRECONDITION_FAILED",
          message: "api is not setup to handle keys",
        });
      }

      const keyId = newId("key");
      const { key, hash, start } = await newKey({
        prefix: input.prefix,
        byteLength: input.bytesLength,
      });

      await db.transaction(async (tx) => {
        await tx.insert(schema.keys).values({
          id: keyId,
          keyAuthId: api.keyAuthId!,
          name: input.name,
          hash,
          start,
          ownerId: input.ownerId,
          meta: JSON.stringify(input.meta ?? {}),
          workspaceId: workspace.id,
          forWorkspaceId: null,
          expires: input.expires ? new Date(input.expires) : null,
          createdAt: new Date(),
          ratelimitLimit: input.ratelimit?.limit,
          ratelimitRefillRate: input.ratelimit?.refillRate,
          ratelimitRefillInterval: input.ratelimit?.refillInterval,
          ratelimitType: input.ratelimit?.type,
          remaining: input.remaining,
          totalUses: 0,
          deletedAt: null,
        });
        await tx.insert(schema.auditLogs).values({
          id: newId("auditLog"),
          time: new Date(),
          workspaceId: workspace.id,
          apiId: api.id,
          actorType: "user",
          actorId: ctx.user.id,
          event: "key.create",
          description: `created key ${keyId} for api ${api.id}`,
          keyId: keyId,
        });
      });
      return { keyId, key };
    }),
  createInternalRootKey: t.procedure
    .use(auth)
    .input(
      z
        .object({
          name: z.string().optional(),
        })
        .optional(),
    )
    .mutation(async ({ ctx, input }) => {
      const unkeyApi = await db.query.apis.findFirst({
        where: eq(schema.apis.id, env().UNKEY_API_ID),
        with: {
          workspace: true,
        },
      });
      if (!unkeyApi) {
        throw new TRPCError({ code: "NOT_FOUND", message: `api ${env().UNKEY_API_ID} not found` });
      }
      if (!unkeyApi.keyAuthId) {
        throw new TRPCError({
          code: "PRECONDITION_FAILED",
          message: `api ${env().UNKEY_API_ID} is not setup to handle keys`,
        });
      }

      const workspace = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
        with: {
          apis: {
            columns: {
              id: true,
            },
          },
        },
      });
      if (!workspace) {
        console.error(`workspace for tenant ${ctx.tenant.id} not found`);
        throw new TRPCError({ code: "NOT_FOUND", message: "workspace not found" });
      }

      const keyId = newId("key");
      const { key, hash, start } = await newKey({ prefix: "unkey", byteLength: 16 });
      await db.transaction(async (tx) => {
        await tx.insert(schema.keys).values({
          id: keyId,
          keyAuthId: unkeyApi.keyAuthId!,
          name: input?.name,
          hash,
          start,
          ownerId: ctx.user.id,
          workspaceId: env().UNKEY_WORKSPACE_ID,
          forWorkspaceId: workspace.id,
          expires: null,
          createdAt: new Date(),
          ratelimitLimit: 10,
          ratelimitRefillRate: 10,
          ratelimitRefillInterval: 1000,
          ratelimitType: "fast",
          remaining: null,
          totalUses: 0,
          deletedAt: null,
        });
        await tx.insert(schema.auditLogs).values({
          id: newId("auditLog"),
          time: new Date(),
          workspaceId: workspace.id,
          apiId: unkeyApi.id,
          actorType: "user",
          actorId: ctx.user.id,
          event: "key.create",
          description: `created key ${keyId} for api ${unkeyApi.id}`,
          keyId: keyId,
        });
      });

      const requiredRoles = workspace.apis.flatMap((api) => [
        `${api.id}::read`,
        `${api.id}::update`,
        `${api.id}::listKeys`,
        `${api.id}::keys::create`,
        `${api.id}::keys::read`,
        `${api.id}::keys::update`,
        `${api.id}::keys::delete`,
      ]);

      const existingRoles = await db.query.roles.findMany({
        where: (table, { eq, and }) =>
          and(eq(table.workspaceId, workspace.id), eq(table.apiId, env().UNKEY_API_ID)),
      });

      const missingRoles = requiredRoles
        .filter((role) => !existingRoles.find((existingRole) => existingRole.name === role))
        .map((name) => {
          return {
            id: newId("role"),
            workspaceId: workspace.id,
            apiId: env().UNKEY_API_ID,
            name,
          };
        });

      if (missingRoles.length > 0) {
        await db.insert(schema.roles).values(missingRoles);
      }

      const allRoles = [...existingRoles, ...missingRoles];

      await db.insert(schema.rolesToKeys).values(
        allRoles.map((role) => ({
          keyId,
          roleId: role.id,
        })),
      );

      return { key, keyId };
    }),
  delete: t.procedure
    .use(auth)
    .input(
      z.object({
        keyIds: z.array(z.string()),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const workspace = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      });
      if (!workspace) {
        throw new TRPCError({ code: "NOT_FOUND", message: "workspace not found" });
      }

      await Promise.all(
        input.keyIds.map(async (keyId) => {
          const key = await db.query.keys.findFirst({
            where: (table, { eq, and }) =>
              and(eq(table.id, keyId), eq(table.workspaceId, workspace.id)),
            with: {
              keyAuth: {
                with: {
                  api: true,
                },
              },
            },
          });
          if (!key) {
            console.warn(`key ${keyId} not found, skipping deletion`);
            return;
          }
          if (key.deletedAt !== null) {
            console.warn(`key ${keyId} already deleted, skipping deletion`);
            return;
          }
          await db.transaction(async (tx) => {
            await db
              .update(schema.keys)
              .set({
                deletedAt: new Date(),
              })
              .where(eq(schema.keys.id, keyId));
            await tx.insert(schema.auditLogs).values({
              id: newId("auditLog"),
              time: new Date(),
              workspaceId: workspace.id,
              apiId: key.keyAuth.api?.id,
              actorType: "user",
              actorId: ctx.user.id,
              event: "key.delete",
              description: `deleted key ${keyId}`,

              keyId: keyId,
            });
          });
        }),
      );
      return;
    }),
  deleteRootKey: t.procedure
    .use(auth)
    .input(
      z.object({
        keyIds: z.array(z.string()),
      }),
    )
    .mutation(async ({ ctx, input }) => {
      const workspace = await db.query.workspaces.findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      });
      if (!workspace) {
        throw new TRPCError({ code: "NOT_FOUND", message: "workspace not found" });
      }

      await Promise.all(
        input.keyIds.map(async (keyId) => {
          const key = await db.query.keys.findFirst({
            where: (table, { eq, and }) =>
              and(eq(table.id, keyId), eq(table.forWorkspaceId, workspace.id)),
          });
          if (!key) {
            console.warn(`key ${keyId} not found, skipping deletion`);
            return;
          }
          if (key.deletedAt !== null) {
            console.warn(`key ${keyId} already deleted, skipping deletion`);
            return;
          }
          await db
            .update(schema.keys)
            .set({
              deletedAt: new Date(),
            })
            .where(eq(schema.keys.id, keyId));
        }),
      );
      return;
    }),
});
