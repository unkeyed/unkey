import { VaultService } from "@/gen/proto/vault/v1/service_pb";
import { insertAuditLogs } from "@/lib/audit";
import { and, db, eq, inArray, schema } from "@/lib/db";
import { createVaultClient } from "@/lib/vault-client";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { ratelimit, withRatelimit, workspaceProcedure } from "../../../trpc";
import {
  type AddEnvVarsResult,
  type EncryptedVariable,
  addEnvVarsInput,
  envVarRows,
} from "./add-plan";
import { envVarAuditLogs, lockAndFindTakenKeys, toEnvVarWriteError } from "./env-var-write";

const vault = createVaultClient(VaultService);

export const addEnvVars = workspaceProcedure
  .use(withRatelimit(ratelimit.create))
  .input(addEnvVarsInput)
  .mutation(async ({ ctx, input }): Promise<AddEnvVarsResult> => {
    const environmentIds = [...new Set(input.environmentIds)];
    const keys = input.variables.map((v) => v.key);

    try {
      const environments = await db.query.environments.findMany({
        where: and(
          eq(schema.environments.workspaceId, ctx.workspace.id),
          eq(schema.environments.appId, input.appId),
          inArray(schema.environments.id, environmentIds),
        ),
        columns: { id: true, slug: true },
      });
      if (environments.length !== environmentIds.length) {
        throw new TRPCError({ code: "NOT_FOUND", message: "Environment not found" });
      }

      const encrypted = await Promise.all(
        environments.map(async (environment) => {
          const variables = input.variables.map((v) => ({
            ...v,
            id: newId("environmentVariable"),
          }));
          const result = await vault.encryptBulk({
            keyring: environment.id,
            items: Object.fromEntries(variables.map((v) => [v.id, v.value])),
          });
          return envVarRows({
            workspaceId: ctx.workspace.id,
            appId: input.appId,
            environmentId: environment.id,
            variables: variables.map(
              (v): EncryptedVariable => ({ ...v, ciphertext: result.items[v.id].encrypted }),
            ),
          });
        }),
      );

      return await db.transaction(async (tx) => {
        const taken = await lockAndFindTakenKeys(tx, {
          appId: input.appId,
          environmentIds,
          keys,
        });
        if (taken.length > 0) {
          return { status: "taken", keys: taken };
        }

        await tx.insert(schema.appEnvironmentVariables).values(encrypted.flat());
        await insertAuditLogs(
          tx,
          envVarAuditLogs({
            workspaceId: ctx.workspace.id,
            userId: ctx.user.id,
            context: { location: ctx.audit.location, userAgent: ctx.audit.userAgent },
            changes: environments.flatMap((environment) =>
              keys.map((key) => ({ action: "set" as const, environment, key })),
            ),
          }),
        );
        return { status: "added", count: keys.length };
      });
    } catch (error) {
      throw toEnvVarWriteError(error, "Failed to add environment variables");
    }
  });
