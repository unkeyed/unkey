import { createKeyInputSchema } from "@/app/(app)/apis/[apiId]/_components/create-key/create-key.schema";
import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireUser, t } from "../../trpc";
import { createApiCore } from "../api/create";
import { createKeyCore } from "../key/create";
import { createWorkspaceCore } from "../workspace/create";

const createWorkspaceWithApiAndKeyInputSchema = z.object({
  workspaceName: z
    .string()
    .trim()
    .min(3, "Workspace name must be at least 3 characters")
    .max(50, "Workspace name must not exceed 50 characters"),
  apiName: z
    .string()
    .min(3, "API name must be at least 3 characters")
    .max(50, "API name must not exceed 50 characters"),
  ...createKeyInputSchema.omit({ keyAuthId: true }).shape,
});

export const onboardingKeyCreation = t.procedure
  .use(requireUser)
  .input(createWorkspaceWithApiAndKeyInputSchema)
  .mutation(async ({ input, ctx }) => {
    const { workspaceName, apiName, ...keyInput } = input;

    try {
      return await db.transaction(async (tx) => {
        //  Create workspace first
        const workspaceResult = await createWorkspaceCore({ name: workspaceName }, ctx, tx);

        // Create workspace context for API and key creation
        const workspaceCtx = {
          ...ctx,
          workspace: { id: workspaceResult.workspace.id },
        };

        // Create API
        const apiResult = await createApiCore({ name: apiName }, workspaceCtx, tx);

        // Create key using the keyAuthId from the API
        const keyAuth = {
          id: apiResult.keyAuthId,
          storeEncryptedKeys: false, // Default for new APIs. Can be activated by unkey with a support ticket.
        };

        const keyResult = await createKeyCore(
          {
            ...keyInput,
            keyAuthId: apiResult.keyAuthId,
          },
          workspaceCtx,
          tx,
          keyAuth,
        );

        return {
          workspace: workspaceResult.workspace,
          organizationId: workspaceResult.organizationId,
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
