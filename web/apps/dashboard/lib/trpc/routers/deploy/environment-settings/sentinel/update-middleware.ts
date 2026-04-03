import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { appRuntimeSettings } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

export type SentinelConfig = {
  policies: {
    id: string;
    name: string;
    enabled: boolean;
    keyauth: { keySpaceIds: string[] };
  }[];
};

// This is 100% not how we will do it later and is just a shortcut to use keyspace middleware before building the actual UI for it.
export const updateMiddleware = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      keyspaceIds: z.array(z.string()).max(10),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const sentinelConfig: SentinelConfig = {
      policies: [],
    };
    if (input.keyspaceIds.length > 0) {
      const keyspaces = await db.query.keyAuth
        .findMany({
          where: {
            workspaceId: ctx.workspace.id,
            RAW: (table, { inArray }) => inArray(table.id, input.keyspaceIds),
          },
          columns: { id: true },
        })
        .catch((err) => {
          console.error(err);
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: "unable to load keyspaces",
          });
        });

      for (const id of input.keyspaceIds) {
        if (!keyspaces.find((ks) => ks.id === id)) {
          throw new TRPCError({
            code: "NOT_FOUND",
            message: `keyspace ${id} does not exist`,
          });
        }
      }

      sentinelConfig.policies.push({
        id: "keyauth-policy",
        name: "API Key Auth",
        enabled: true,
        keyauth: { keySpaceIds: keyspaces.map((ks) => ks.id) },
      });
    }

    const env = await db.query.environments.findFirst({
      where: { id: input.environmentId, workspaceId: ctx.workspace.id },
      columns: { appId: true },
    });
    if (!env) {
      throw new TRPCError({ code: "NOT_FOUND", message: "Environment not found" });
    }

    const serialized = JSON.stringify(sentinelConfig);

    await db
      .insert(appRuntimeSettings)
      .values({
        workspaceId: ctx.workspace.id,
        appId: env.appId,
        environmentId: input.environmentId,
        sentinelConfig: serialized,
        createdAt: Date.now(),
        updatedAt: Date.now(),
      })
      .onDuplicateKeyUpdate({ set: { sentinelConfig: serialized, updatedAt: Date.now() } });
  });
