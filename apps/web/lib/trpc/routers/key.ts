import { db, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { auth, t } from "../trpc";

import { unkeyRoot, unkeyScoped } from "@/lib/api";
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
        where: eq(schema.workspaces.tenantId, ctx.tenant.id),
      });
      if (!workspace) {
        throw new TRPCError({ code: "NOT_FOUND", message: "workspace not found" });
      }

      const newRootKey = await unkeyRoot._internal.createRootKey({
        forWorkspaceId: workspace.id,
        name: "Dashboard",
        expires: Date.now() + 60000, // expires in 1 minute
      });
      if (newRootKey.error) {
        throw new TRPCError({ code: "INTERNAL_SERVER_ERROR", message: newRootKey.error.message });
      }

      const { error, result } = await unkeyScoped(newRootKey.result.key).keys.create({
        apiId: input.apiId,
        name: input.name ?? undefined,
        prefix: input.prefix,
        byteLength: input.bytesLength,
        ownerId: input.ownerId ?? undefined,
        meta: input.meta,
        remaining: input.remaining,
        expires: input.expires ?? undefined,
        ratelimit: input.ratelimit,
      });
      if (error) {
        throw new TRPCError({ code: "INTERNAL_SERVER_ERROR", message: error.message });
      }
      return result;
    }),
  createInternalRootKey: t.procedure.use(auth).mutation(async ({ ctx }) => {
    const unkeyApi = await db.query.apis.findFirst({
      where: eq(schema.apis.id, env().UNKEY_API_ID),
      with: {
        workspace: true,
      },
    });
    if (!unkeyApi) {
      console.error(`api ${env().UNKEY_API_ID} not found`);
      throw new TRPCError({ code: "NOT_FOUND", message: `api ${env().UNKEY_API_ID} not found` });
    }

    const workspace = await db.query.workspaces.findFirst({
      where: eq(schema.workspaces.tenantId, ctx.tenant.id),
    });
    if (!workspace) {
      console.error(`workspace for tenant ${ctx.tenant.id} not found`);
      throw new TRPCError({ code: "NOT_FOUND", message: "workspace not found" });
    }

    const { error, result } = await unkeyRoot._internal.createRootKey({
      forWorkspaceId: workspace.id,
    });

    if (error) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: `unable to create root key: ${error.message}`,
      });
    }

    return result;
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
        where: eq(schema.workspaces.tenantId, ctx.tenant.id),
      });
      if (!workspace) {
        throw new TRPCError({ code: "NOT_FOUND", message: "workspace not found" });
      }

      const newRootKey = await unkeyRoot._internal.createRootKey({
        forWorkspaceId: workspace.id,
        name: "Dashboard",
        expires: Date.now() + 60000, // expires in 1 minute
      });
      if (newRootKey.error) {
        throw new TRPCError({ code: "INTERNAL_SERVER_ERROR", message: newRootKey.error.message });
      }

      const sdk = unkeyScoped(newRootKey.result.key);

      await Promise.all(
        input.keyIds.map(async (keyId) => {
          const { error } = await sdk.keys.revoke({
            keyId,
          });
          if (error) {
            throw new TRPCError({ code: "INTERNAL_SERVER_ERROR", message: error.message });
          }
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
        where: eq(schema.workspaces.tenantId, ctx.tenant.id),
      });
      if (!workspace) {
        throw new TRPCError({ code: "NOT_FOUND", message: "workspace not found" });
      }

      const newRootKey = await unkeyRoot._internal.createRootKey({
        forWorkspaceId: workspace.id,
        name: "Dashboard",
        expires: Date.now() + 60000, // expires in 1 minute
      });
      if (newRootKey.error) {
        throw new TRPCError({ code: "INTERNAL_SERVER_ERROR", message: newRootKey.error.message });
      }

      const sdk = unkeyScoped(newRootKey.result.key);

      await Promise.all(
        input.keyIds.map(async (keyId) => {
          const { error } = await sdk._internal.deleteRootKey({
            keyId,
          });
          if (error) {
            console.log(error);
            throw new TRPCError({ code: "INTERNAL_SERVER_ERROR", message: error.message });
          }
        }),
      );
      return;
    }),
});
