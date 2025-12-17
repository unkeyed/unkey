import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { insertAuditLogs } from "@/lib/audit";
import { db, schema } from "@/lib/db";
import { DatabaseError } from "@planetscale/database";
import { newId } from "@unkey/id";
import { workspaceProcedure } from "../../trpc";
export const createNamespace = workspaceProcedure
  .input(
    z.object({
      name: z.string().min(1).max(50),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const namespaceId = newId("ratelimitNamespace");
    await db
      .transaction(async (tx) => {
        await tx.insert(schema.ratelimitNamespaces).values({
          id: namespaceId,
          name: input.name,
          workspaceId: ctx.workspace.id,

          createdAtM: Date.now(),
        });
        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: "ratelimitNamespace.create",
          description: `Created ${namespaceId}`,
          resources: [
            {
              type: "ratelimitNamespace",
              id: namespaceId,
              name: input.name,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      })
      .catch((e) => {
        console.error("Failed to create namespace", e);
        if (e instanceof DatabaseError && e.body.message.includes("Duplicate entry")) {
          throw new TRPCError({
            code: "CONFLICT",
            message: `A namespace with name "${input.name}" already exists in this workspace. Please choose a different name.`,
          });
        }
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to create namespace. Please try again or contact support@unkey.dev",
        });
      });

    return {
      id: namespaceId,
    };
  });
