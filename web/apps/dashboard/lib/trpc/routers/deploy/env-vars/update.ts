import { and, db, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { Vault } from "@/lib/vault";
import { TRPCError } from "@trpc/server";
import { environments } from "@unkey/db/src/schema";
import { envVarKeySchema } from "@/lib/schemas/env-var";
import { z } from "zod";
import { workspaceProcedure } from "../../../trpc";

const vault = new Vault({
  baseUrl: env().VAULT_URL,
  token: env().VAULT_TOKEN,
});

export const updateEnvVar = workspaceProcedure
  .input(
    z.object({
      envVarId: z.string(),
      // Key can only be updated for recoverable vars (validated on client)
      key: envVarKeySchema.optional(),
      environmentId: z.string().trim().min(1, "Environment is required"),
      // Value is always re-encrypted
      value: z.string().min(1),
      type: z.enum(["recoverable", "writeonly"]),
      description: z.string().nullable().optional(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    try {
      const envVar = await db.query.appEnvironmentVariables.findFirst({
        where: and(
          eq(schema.appEnvironmentVariables.id, input.envVarId),
          eq(schema.appEnvironmentVariables.workspaceId, ctx.workspace.id),
        ),
        columns: {
          id: true,
          type: true,
          key: true,
          environmentId: true,
        },
      });

      if (!envVar) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Environment variable not found",
        });
      }

      if (envVar.type === "writeonly" && input.key && input.key !== envVar.key) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "Cannot rename writeonly environment variables",
        });
      }

      if (input.type !== envVar.type) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "Cannot change environment variable type after creation",
        });
      }

      const [currentEnvironment, targetEnvironment] = await Promise.all([
        db.query.environments.findFirst({
          where: and(
            eq(environments.id, envVar.environmentId),
            eq(environments.workspaceId, ctx.workspace.id),
          ),
          columns: {
            id: true,
            appId: true,
          },
        }),
        db.query.environments.findFirst({
          where: and(
            eq(environments.id, input.environmentId),
            eq(environments.workspaceId, ctx.workspace.id),
          ),
          columns: {
            id: true,
            appId: true,
          },
        }),
      ]);

      if (!currentEnvironment) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Current environment not found",
        });
      }

      if (!targetEnvironment) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Environment not found",
        });
      }

      if (targetEnvironment.appId !== currentEnvironment.appId) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "Cannot move environment variable to a different app",
        });
      }

      const { encrypted } = await vault.encrypt({
        keyring: input.environmentId,
        data: input.value,
      });

      await db
        .update(schema.appEnvironmentVariables)
        .set({
          key: input.key ?? envVar.key,
          value: encrypted,
          type: input.type,
          environmentId: input.environmentId,
          ...(input.description !== undefined ? { description: input.description } : {}),
        })
        .where(eq(schema.appEnvironmentVariables.id, input.envVarId));
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to update environment variable",
      });
    }
  });
