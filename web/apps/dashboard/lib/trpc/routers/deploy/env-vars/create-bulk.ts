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

    const variablesWithIds = input.variables.map((v) => {
      const environment = envMap.get(v.environmentId);
      if (!environment) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: `Environment ${v.environmentId} not found`,
        });
      }
      return { ...v, id: newId("environmentVariable"), appId: environment.appId };
    });

    // Group by keyring (environmentId) so each encryptBulk call uses one DEK, avoids concurrent S3 PutObject on the same key
    const grouped = Map.groupBy(variablesWithIds, (v) => v.environmentId);

    const allEncrypted = new Map<string, string>();
    await Promise.all(
      grouped.entries().map(async ([environmentId, vars]) => {
        const items = Object.fromEntries(vars.map((v) => [v.id, v.value]));
        const result = await vault.encryptBulk({ keyring: environmentId, items });
        for (const [id, item] of Object.entries(result.items)) {
          allEncrypted.set(id, item.encrypted);
        }
      }),
    );

    const encryptedVars = variablesWithIds.map((v) => {
      const encrypted = allEncrypted.get(v.id);
      if (!encrypted) {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: `Missing encryption result for variable ${v.id}`,
        });
      }
      return {
        id: v.id,
        workspaceId: ctx.workspace.id,
        appId: v.appId,
        environmentId: v.environmentId,
        key: v.key,
        value: encrypted,
        type: v.type,
        description: v.description ?? null,
      };
    });

    await db.insert(schema.appEnvironmentVariables).values(encryptedVars);
  });
