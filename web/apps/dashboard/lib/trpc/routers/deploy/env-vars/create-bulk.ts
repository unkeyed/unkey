import { VaultService } from "@/gen/proto/vault/v1/service_pb";
import { and, db, eq, inArray, schema } from "@/lib/db";
import { envVarKeySchema, envVarValueSchema } from "@/lib/schemas/env-var";
import { createVaultClient } from "@/lib/vault-client";
import { TRPCError } from "@trpc/server";
import { environments } from "@unkey/db/src/schema";
import { newId } from "@unkey/id";
import { z } from "zod";
import { ratelimit, withRatelimit, workspaceProcedure } from "../../../trpc";

const vault = createVaultClient(VaultService);

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

    // Group by keyring (environmentId) so each encryptBulk call uses one DEK, avoids concurrent S3 PutObject on the same key
    const grouped = Map.groupBy(input.variables, (v) => v.environmentId);

    const encryptedVars = (
      await Promise.all(
        grouped.entries().map(async ([environmentId, vars]) => {
          const environment = envMap.get(environmentId);
          if (!environment) {
            throw new TRPCError({
              code: "NOT_FOUND",
              message: `Environment ${environmentId} not found`,
            });
          }

          const tagged = vars.map((v) => [newId("environmentVariable"), v] as const);
          const items = Object.fromEntries(tagged.map(([id, v]) => [id, v.value]));
          const result = await vault.encryptBulk({ keyring: environmentId, items });

          return tagged.map(([id, v]) => ({
            id,
            workspaceId: ctx.workspace.id,
            appId: environment.appId,
            environmentId,
            key: v.key,
            value: result.items[id].encrypted,
            type: v.type,
            description: v.description ?? null,
          }));
        }),
      )
    ).flat();

    await db.insert(schema.appEnvironmentVariables).values(encryptedVars);
  });
