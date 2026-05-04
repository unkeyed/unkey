import { and, db, eq, inArray, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { envVarKeySchema, envVarValueSchema } from "@/lib/schemas/env-var";
import { Vault } from "@/lib/vault";
import { TRPCError } from "@trpc/server";
import { environments } from "@unkey/db/src/schema";
import { newId } from "@unkey/id";
import { z } from "zod";
import { workspaceProcedure } from "../../../trpc";

const vault = new Vault({
  baseUrl: env().VAULT_URL,
  token: env().VAULT_TOKEN,
});

const bulkEnvVarInputSchema = z.object({
  environmentId: z.string(),
  key: envVarKeySchema,
  value: envVarValueSchema,
  type: z.enum(["recoverable", "writeonly"]),
  description: z.string().nullable().optional(),
});

export const createBulkEnvVars = workspaceProcedure
  .input(
    z.object({
      variables: z.array(bulkEnvVarInputSchema).min(1),
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

          const ids = vars.map(() => newId("environmentVariable"));
          const items = Object.fromEntries(ids.map((id, i) => [id, vars[i].value]));
          const result = await vault.encryptBulk({ keyring: environmentId, items });

          return vars.map((v, i) => ({
            id: ids[i],
            workspaceId: ctx.workspace.id,
            appId: environment.appId,
            environmentId,
            key: v.key,
            value: result.items[ids[i]].encrypted,
            type: v.type,
            description: v.description ?? null,
          }));
        }),
      )
    ).flat();

    await db.insert(schema.appEnvironmentVariables).values(encryptedVars);
  });
