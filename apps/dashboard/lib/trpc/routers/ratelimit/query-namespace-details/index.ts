import { db } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

const workspaceDetailsInput = z.object({
  namespaceId: z.string(),
  includeOverrides: z.boolean().optional().default(false),
});

const namespaceSchema = z.object({
  id: z.string(),
  workspaceId: z.string(),
  name: z.string(),
  overrides: z
    .array(
      z.object({
        id: z.string(),
        identifier: z.string(),
        limit: z.number(),
        duration: z.number(),
        async: z.boolean(),
      }),
    )
    .optional(),
});

const workspaceDetailsOutput = z.object({
  namespace: namespaceSchema,
  ratelimitNamespaces: z.array(namespaceSchema),
});

export type WorkspaceDetailsResponse = z.infer<typeof workspaceDetailsOutput>;

export const queryRatelimitWorkspaceDetails = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(workspaceDetailsInput)
  .output(workspaceDetailsOutput)
  .query(async ({ ctx, input }) => {
    try {
      const result = await fetchWorkspaceDetails({
        orgId: ctx.workspace.orgId,
        namespaceId: input.namespaceId,
        includeOverrides: input.includeOverrides,
      });
      return result;
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      console.error("Failed to fetch workspace details", JSON.stringify(error));
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch workspace details",
      });
    }
  });

export type WorkspaceDetailsOptions = {
  orgId: string;
  namespaceId: string;
  includeOverrides?: boolean;
};

export async function fetchWorkspaceDetails({
  orgId,
  namespaceId,
  includeOverrides = false,
}: WorkspaceDetailsOptions): Promise<WorkspaceDetailsResponse> {
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
    columns: {
      name: true,
      orgId: true,
    },
    with: {
      ratelimitNamespaces: {
        where: (table, { isNull }) => isNull(table.deletedAtM),
        columns: {
          id: true,
          workspaceId: true,
          name: true,
        },
        with: includeOverrides
          ? {
              overrides: {
                where: (table, { isNull }) => isNull(table.deletedAtM),
                columns: {
                  id: true,
                  identifier: true,
                  limit: true,
                  duration: true,
                  async: true,
                },
              },
            }
          : undefined,
      },
    },
  });

  if (!workspace) {
    throw new TRPCError({
      code: "NOT_FOUND",
      message: "Workspace not found",
    });
  }

  const namespace = workspace.ratelimitNamespaces.find((ns) => ns.id === namespaceId);

  if (!namespace) {
    throw new TRPCError({
      code: "NOT_FOUND",
      message: "Namespace not found in workspace",
    });
  }

  const result: WorkspaceDetailsResponse = {
    namespace,
    ratelimitNamespaces: workspace.ratelimitNamespaces,
  };

  return result;
}
