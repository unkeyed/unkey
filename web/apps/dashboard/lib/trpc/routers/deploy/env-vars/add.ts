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
  envVarAuditLogs,
  envVarRows,
  takenKeys,
} from "./add-plan";

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
        await tx
          .select({ id: schema.environments.id })
          .from(schema.environments)
          .where(inArray(schema.environments.id, environmentIds))
          .for("update");

        const existing = await tx
          .select({ key: schema.appEnvironmentVariables.key })
          .from(schema.appEnvironmentVariables)
          .where(
            and(
              eq(schema.appEnvironmentVariables.appId, input.appId),
              inArray(schema.appEnvironmentVariables.environmentId, environmentIds),
              inArray(schema.appEnvironmentVariables.key, keys),
            ),
          );
        if (existing.length > 0) {
          return { status: "taken", keys: takenKeys(existing) };
        }

        await tx.insert(schema.appEnvironmentVariables).values(encrypted.flat());
        await insertAuditLogs(
          tx,
          envVarAuditLogs({
            workspaceId: ctx.workspace.id,
            userId: ctx.user.id,
            context: { location: ctx.audit.location, userAgent: ctx.audit.userAgent },
            environments,
            keys,
          }),
        );
        return { status: "added", count: keys.length };
      });
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      console.error("Failed to add environment variables", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to add environment variables",
      });
    }
  });
