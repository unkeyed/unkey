import { and, db, eq } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { appRuntimeSettings, apps } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

// This is 100% not how we will do it later and is just a shortcut to use keyspace middleware before building the actual UI for it.
export const updateMiddleware = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      keyspaceIds: z.array(z.string()).max(10),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const app = await db.query.apps.findFirst({
      where: and(eq(apps.workspaceId, ctx.workspace.id)),
      columns: { id: true },
    });
    if (!app) {
      return;
    }

    const sentinelConfig: {
      policies: {
        id: string;
        name: string;
        enabled: boolean;
        keyauth: { keySpaceIds: string[] };
      }[];
    } = {
      policies: [],
    };
    if (input.keyspaceIds.length > 0) {
      const keyspaces = await db.query.keyAuth
        .findMany({
          where: (table, { and, inArray }) =>
            and(inArray(table.id, input.keyspaceIds), eq(table.workspaceId, ctx.workspace.id)),
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
        keyauth: { keySpaceIds: ["ks_NNh4XwVsZiwG"] },
      });
    }
    await db
      .update(appRuntimeSettings)
      .set({ sentinelConfig: JSON.stringify(sentinelConfig) })
      .where(
        and(
          eq(appRuntimeSettings.workspaceId, ctx.workspace.id),
          eq(appRuntimeSettings.appId, app.id),
          eq(appRuntimeSettings.environmentId, input.environmentId),
        ),
      );
  });
