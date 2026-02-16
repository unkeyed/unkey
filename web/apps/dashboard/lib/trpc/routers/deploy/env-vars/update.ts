import { and, db, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { Vault } from "@/lib/vault";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../../trpc";

const vault = new Vault({
  baseUrl: env().VAULT_URL,
  token: env().VAULT_TOKEN,
});

export const updateEnvVar = workspaceProcedure
  .input(
    z.object({
      envVarId: z.string(),
      // Key can only be updated for recoverable vars (validated on client)
      key: z.string().min(1).optional(),
      // Value is always re-encrypted
      value: z.string().min(1),
      type: z.enum(["recoverable", "writeonly"]),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    try {
      const envVar = await db.query.environmentVariables.findFirst({
        where: and(
          eq(schema.environmentVariables.id, input.envVarId),
          eq(schema.environmentVariables.workspaceId, ctx.workspace.id),
        ),
        columns: {
          id: true,
          type: true,
          key: true,
          environmentId: true,
        },
      });

      if (!envVar) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Environment variable not found",
        });
      }

      if (envVar.type === "writeonly" && input.key && input.key !== envVar.key) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "Cannot rename writeonly environment variables",
        });
      }

      if (input.type !== envVar.type) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "Cannot change environment variable type after creation",
        });
      }

      const { encrypted } = await vault.encrypt({
        keyring: envVar.environmentId,
        data: input.value,
      });

      await db
        .update(schema.environmentVariables)
        .set({
          key: input.key ?? envVar.key,
          value: encrypted,
          type: input.type,
        })
        .where(eq(schema.environmentVariables.id, input.envVarId));
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to update environment variable",
      });
    }
  });
