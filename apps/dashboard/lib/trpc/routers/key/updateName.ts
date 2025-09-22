import { nameSchema } from "@/app/(app)/[workspace]/apis/[apiId]/_components/create-key/create-key.schema";
import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";

export const updateKeyName = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(
    z.object({
      keyId: z.string(),
      name: nameSchema,
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const key = await db.query.keys
      .findFirst({
        where: (table, { eq, isNull, and }) =>
          and(
            eq(table.workspaceId, ctx.workspace.id),
            eq(table.id, input.keyId),
            isNull(table.deletedAtM),
          ),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to update the name on this key. Please try again or contact support@unkey.dev",
        });
      });

    if (!key) {
      throw new TRPCError({
        message:
          "We are unable to find the correct key. Please try again or contact support@unkey.dev.",
        code: "NOT_FOUND",
      });
    }

    // Normalize both values
    const normalizedNewName = (input.name || "").trim();
    const normalizedCurrentName = (key.name || "").trim();

    // Check if the name is actually changing
    if (normalizedNewName === normalizedCurrentName && normalizedNewName !== "") {
      throw new TRPCError({
        message: "New name must be different from the current name",
        code: "UNPROCESSABLE_CONTENT",
      });
    }

    await db
      .transaction(async (tx) => {
        await tx
          .update(schema.keys)
          .set({
            name: input.name ?? null,
          })
          .where(eq(schema.keys.id, key.id))
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "We are unable to update name on this key. Please try again or contact support@unkey.dev",
            });
          });
        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: "key.update",
          description: `Changed name of ${key.id} to ${input.name}`,
          resources: [
            {
              type: "key",
              id: key.id,
              name: input.name ?? undefined,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to update name on this key. Please try again or contact support@unkey.dev",
        });
      });

    return {
      keyId: key.id,
      updated: true,
      previousName: key.name,
      newName: input.name ?? null,
    };
  });
