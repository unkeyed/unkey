import { and, db, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { envVarKeySchema, envVarValueSchema } from "@/lib/schemas/env-var";
import { Vault } from "@/lib/vault";
import { TRPCError } from "@trpc/server";
import { environments } from "@unkey/db/src/schema";
import { newId } from "@unkey/id";
import { z } from "zod";
import { ratelimit, withRatelimit, workspaceProcedure } from "../../../trpc";

const vault = new Vault({
  baseUrl: env().VAULT_URL,
  token: env().VAULT_TOKEN,
});

// Cap per-request fanout into Vault encrypt + DB inserts to stop authenticated
// users from amplifying one tRPC call into thousands of internal requests.
const MAX_ENV_VARS_PER_REQUEST = 100;

const envVarInputSchema = z.object({
  key: envVarKeySchema,
  value: envVarValueSchema,
  type: z.enum(["recoverable", "writeonly"]),
  description: z.string().nullable().optional(),
});

export const createEnvVars = workspaceProcedure
  .use(withRatelimit(ratelimit.create))
  .input(
    z.object({
      environmentId: z.string(),
      variables: z.array(envVarInputSchema).min(1).max(MAX_ENV_VARS_PER_REQUEST),
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

      const encryptedVars = await Promise.all(
        input.variables.map(async (v) => {
          const { encrypted } = await vault.encrypt({
            keyring: input.environmentId,
            data: v.value,
          });

          return {
            id: newId("environmentVariable"),
            workspaceId: ctx.workspace.id,
            appId: environment.appId,
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
