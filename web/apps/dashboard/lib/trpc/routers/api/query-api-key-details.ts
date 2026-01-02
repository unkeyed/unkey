import { and, db, eq, isNull } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { apis } from "@unkey/db/src/schema";
import { z } from "zod";

const queryApiLayoutPayload = z.object({
  apiId: z.string().min(1, "API ID is required"),
});

const apiLayoutResponse = z.object({
  currentApi: z.object({
    id: z.string(),
    name: z.string(),
    workspaceId: z.string(),
    keyAuthId: z.string().nullable(),
    keyspaceDefaults: z
      .object({
        prefix: z.string().optional(),
        bytes: z.number().optional(),
      })
      .nullable(),
    deleteProtection: z.boolean().nullable(),
    ipWhitelist: z.string().nullable(),
  }),
  workspaceApis: z.array(
    z.object({
      id: z.string(),
      name: z.string(),
    }),
  ),
  keyAuth: z
    .object({
      id: z.string(),
      defaultPrefix: z.string().nullable(),
      defaultBytes: z.number().nullable(),
      sizeApprox: z.number(),
    })
    .nullable(),
  workspace: z.object({
    id: z.string(),
    ipWhitelist: z.boolean(),
  }),
});

export const queryApiKeyDetails = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(queryApiLayoutPayload)
  .output(apiLayoutResponse)
  .query(async ({ ctx, input }) => {
    const { apiId } = input;

    try {
      const currentApi = await db.query.apis.findFirst({
        where: (table) =>
          and(
            eq(table.id, apiId),
            isNull(table.deletedAtM),
            eq(table.workspaceId, ctx.workspace.id),
          ),
        with: {
          workspace: {
            columns: {
              id: true,
              orgId: true,
              features: true,
            },
          },
          keyAuth: {
            columns: {
              id: true,
              defaultPrefix: true,
              defaultBytes: true,
              sizeApprox: true,
            },
          },
        },
        columns: {
          id: true,
          name: true,
          workspaceId: true,
          keyAuthId: true,
          deleteProtection: true,
          ipWhitelist: true,
        },
      });

      if (!currentApi) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: `API ${apiId} not found or access denied`,
        });
      }

      // Fetch all APIs in the same workspace
      const workspaceApis = await db
        .select({ id: apis.id, name: apis.name })
        .from(apis)
        .where(and(eq(apis.workspaceId, currentApi.workspaceId), isNull(apis.deletedAtM)))
        .orderBy(apis.name);

      return {
        currentApi: {
          id: currentApi.id,
          name: currentApi.name,
          workspaceId: currentApi.workspaceId,
          keyAuthId: currentApi.keyAuthId,
          deleteProtection: currentApi.deleteProtection,
          ipWhitelist: currentApi.ipWhitelist,
          keyspaceDefaults: currentApi.keyAuth
            ? {
                prefix: currentApi.keyAuth.defaultPrefix || undefined,
                bytes: currentApi.keyAuth.defaultBytes || undefined,
              }
            : null,
        },
        workspaceApis,
        keyAuth: currentApi.keyAuth,
        workspace: {
          id: currentApi.workspace.id,
          ipWhitelist: Boolean(currentApi.workspace.features.ipWhitelist),
        },
      };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch API layout data",
      });
    }
  });
