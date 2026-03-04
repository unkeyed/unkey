import { and, db, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { Vault } from "@/lib/vault";
import { TRPCError } from "@trpc/server";
import { apps, environments } from "@unkey/db/src/schema";
import { newId } from "@unkey/id";
import { z } from "zod";
import { workspaceProcedure } from "../../../trpc";

const vault = new Vault({
  baseUrl: env().VAULT_URL,
  token: env().VAULT_TOKEN,
});

const envVarInputSchema = z.object({
  key: z.string().min(1),
  value: z.string().min(1),
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
          projectId: true,
        },
      });

      if (!environment) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Environment not found",
        });
      }

      // Find the app for this environment and project
      const app = await db.query.apps.findFirst({
        where: and(
          eq(apps.projectId, environment.projectId),
          eq(apps.environmentId, input.environmentId),
          eq(apps.workspaceId, ctx.workspace.id),
        ),
        columns: {
          id: true,
        },
      });

      if (!app) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "App not found for this environment",
        });
      }

      const encryptedVars = await Promise.all(
        input.variables.map(async (v) => {
          const { encrypted } = await vault.encrypt({
            keyring: input.environmentId,
            data: v.value,
          });

          return {
            id: newId("environmentVariable"),
            workspaceId: ctx.workspace.id,
            appId: app.id,
            environmentId: input.environmentId,
            key: v.key,
            value: encrypted,
            type: v.type,
            description: v.description ?? null,
          };
        }),
      );

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
