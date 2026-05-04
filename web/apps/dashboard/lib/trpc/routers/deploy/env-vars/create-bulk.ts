import { and, db, eq, inArray, schema } from "@/lib/db";
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
const MAX_BULK_ENV_VARS = 100;

const bulkEnvVarInputSchema = z.object({
  environmentId: z.string(),
  key: envVarKeySchema,
  value: envVarValueSchema,
  type: z.enum(["recoverable", "writeonly"]),
  description: z.string().nullable().optional(),
});

export const createBulkEnvVars = workspaceProcedure
  .use(withRatelimit(ratelimit.create))
  .input(
    z.object({
      variables: z.array(bulkEnvVarInputSchema).min(1).max(MAX_BULK_ENV_VARS),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const environmentIds = [...new Set(input.variables.map((v) => v.environmentId))];

    const envRecords = await db.query.environments.findMany({
      where: and(
        inArray(environments.id, environmentIds),
        eq(environments.workspaceId, ctx.workspace.id),
      ),
      columns: {
        id: true,
        appId: true,
      },
    });

    const envMap = new Map(envRecords.map((e) => [e.id, e]));

    const encryptedVars = await Promise.all(
      input.variables.map(async (v) => {
        const environment = envMap.get(v.environmentId);
        if (!environment) {
          throw new TRPCError({
            code: "NOT_FOUND",
            message: `Environment ${v.environmentId} not found`,
          });
        }
        const { encrypted } = await vault.encrypt({
          keyring: v.environmentId,
          data: v.value,
        });

        return {
          id: newId("environmentVariable"),
          workspaceId: ctx.workspace.id,
          appId: environment.appId,
          environmentId: v.environmentId,
          key: v.key,
          value: encrypted,
          type: v.type,
          description: v.description ?? null,
        };
      }),
    );

    await db.insert(schema.appEnvironmentVariables).values(encryptedVars);
  });
