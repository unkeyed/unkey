import { createKeyInputSchema } from "@/app/(app)/apis/[apiId]/_components/create-key/create-key.schema";
import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";
import { createApiCore } from "../api/create";
import { createKeyCore } from "../key/create";

const createWorkspaceWithApiAndKeyInputSchema = z.object({
  apiName: z
    .string()
    .min(3, "API name must be at least 3 characters")
    .max(50, "API name must not exceed 50 characters"),
  ...createKeyInputSchema.omit({ keyAuthId: true }).shape,
});

export const onboardingKeyCreation = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(createWorkspaceWithApiAndKeyInputSchema)
  .mutation(async ({ input, ctx }) => {
    const { apiName, ...keyInput } = input;

    try {
      return await db.transaction(async (tx) => {
        // Validate that the workspace exists and user has access to it
        const workspace = await tx.query.workspaces
          .findFirst({
            where: (table, { and, eq, isNull }) =>
              and(
                eq(table.id, ctx.workspace.id),
                eq(table.orgId, ctx.tenant.id),
                isNull(table.deletedAtM),
              ),
          })
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "Failed to validate workspace access. If this issue persists, please contact support@unkey.dev",
            });
          });

        if (!workspace) {
          throw new TRPCError({
            code: "NOT_FOUND",
            message: "Workspace not found or you don't have access to it",
          });
        }

        // Create API
        const apiResult = await createApiCore({ name: apiName }, ctx, tx);

        const keyResult = await createKeyCore(
          {
            ...keyInput,
            keyAuthId: apiResult.keyAuthId,
            storeEncryptedKeys: false, // Default for new APIs. Can be activated by unkey with a support ticket.
          },
          ctx,
          tx,
        );

        return {
          apiId: apiResult.id,
          keyId: keyResult.keyId,
          key: keyResult.key,
          keyName: keyResult.name,
        };
      });
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "We are unable to create the workspace, API, and key. Please try again or contact support@unkey.dev",
      });
    }
  });
