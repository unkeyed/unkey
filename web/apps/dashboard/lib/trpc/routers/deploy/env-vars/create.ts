import { VaultService } from "@/gen/proto/vault/v1/service_pb";
import { and, db, eq, schema } from "@/lib/db";
import { envVarKeySchema, envVarValueSchema } from "@/lib/schemas/env-var";
import { createVaultClient } from "@/lib/vault-client";
import { TRPCError } from "@trpc/server";
import { environments } from "@unkey/db/src/schema";
import { newId } from "@unkey/id";
import { z } from "zod";
import { workspaceProcedure } from "../../../trpc";

const vault = createVaultClient(VaultService);

const envVarInputSchema = z.object({
  key: envVarKeySchema,
  value: envVarValueSchema,
  type: z.enum(["recoverable", "writeonly"]),
  description: z.string().nullable().optional(),
});

export const createEnvVars = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      variables: z.array(envVarInputSchema).min(1),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    try {
      const environment = await db.query.environments.findFirst({
        where: and(
          eq(environments.id, input.environmentId),
          eq(environments.workspaceId, ctx.workspace.id),
        ),
        columns: {
          id: true,
          appId: true,
        },
      });

      if (!environment) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Environment not found",
        });
      }

      const variablesWithIds = input.variables.map((v) => ({
        ...v,
        id: newId("environmentVariable"),
      }));

      const items: Record<string, string> = {};
      for (const v of variablesWithIds) {
        items[v.id] = v.value;
      }

      const bulkResult = await vault.encryptBulk({
        keyring: input.environmentId,
        items,
      });

      const encryptedVars = variablesWithIds.map((v) => ({
        id: v.id,
        workspaceId: ctx.workspace.id,
        appId: environment.appId,
        environmentId: input.environmentId,
        key: v.key,
        value: bulkResult.items[v.id].encrypted,
        type: v.type,
        description: v.description ?? null,
      }));

      await db.insert(schema.appEnvironmentVariables).values(encryptedVars);
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to create environment variables",
      });
    }
  });
