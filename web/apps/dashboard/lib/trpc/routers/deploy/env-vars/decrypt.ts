import { and, db, eq } from "@/lib/db";
import { env } from "@/lib/env";
import { createVault } from "@/lib/vault";
import { TRPCError } from "@trpc/server";
import { environmentVariables } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../trpc";

const vault = createVault(env().VAULT_URL, env().VAULT_TOKEN);

export const decryptEnvVar = workspaceProcedure
  .input(
    z.object({
      envVarId: z.string(),
    }),
  )
  .output(
    z.object({
      value: z.string(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    try {
      const envVar = await db.query.environmentVariables.findFirst({
        where: and(
          eq(environmentVariables.id, input.envVarId),
          eq(environmentVariables.workspaceId, ctx.workspace.id),
        ),
        columns: {
          id: true,
          value: true,
          type: true,
          environmentId: true,
        },
      });

      if (!envVar) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Environment variable not found",
        });
      }

      if (envVar.type === "writeonly") {
        throw new TRPCError({
          code: "FORBIDDEN",
          message: "Writeonly environment variables cannot be decrypted",
        });
      }

      const { plaintext } = await vault.decrypt({
        keyring: envVar.environmentId,
        encrypted: envVar.value,
      });

      return { value: plaintext };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to decrypt environment variable",
      });
    }
  });
