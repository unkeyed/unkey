import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";
import { DatabaseError } from "@planetscale/database";
import { newId } from "@unkey/id";
import { auth, t } from "../../trpc";
export const createNamespace = t.procedure
  .use(auth)
  .input(
    z.object({
      name: z.string().min(1).max(50),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const ws = await db.query.workspaces
      .findFirst({
        where: (table, { and, eq, isNull }) =>
          and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to create a new namespace. Please try again or contact support@unkey.dev",
        });
      });
    if (!ws) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct workspace. Please try again or contact support@unkey.dev.",
      });
    }
    const isSoftDeleted = await db.query.ratelimitNamespaces
      .findFirst({
        where: (table, { and, eq }) => and(eq(table.name, input.name)),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to create a new namespace. Please try again or contact support@unkey.dev",
        });
      });

    if (isSoftDeleted) {
      const namespace = await db.query.ratelimitNamespaces
        .findFirst({
          where: eq(schema.ratelimitNamespaces.name, input.name),
        })
        .catch((_err) => {
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message:
              "We were unable to find the namespace. Please try again or contact support@unkey.dev.",
          });
        });
      if (!namespace) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message:
            "We are unable to find the correct namespace. Please try again or contact support@unkey.dev.",
        });
      }
      await db.transaction(async (tx) => {
        await tx
          .update(schema.ratelimitNamespaces)
          .set({
            deletedAt: null,
          })
          .where(eq(schema.ratelimitNamespaces.name, input.name))
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "We were unable to update the namespace. Please try again or contact support@unkey.dev.",
            });
          });

        // const overrides = await tx.query.ratelimitOverrides.findMany({
        //     where: (table, { eq }) => eq(table.namespaceId, namespace.id),
        //     columns: { id: true },
        // });
        // if (overrides.length > 0) {
        //   await tx
        //     .update(schema.ratelimitOverrides)
        //     .set({ deletedAt: null })
        //     .where(eq(schema.ratelimitOverrides.namespaceId, namespace.id))
        //     .catch((_err) => {
        //       throw new TRPCError({
        //         code: "INTERNAL_SERVER_ERROR",
        //         message:
        //           "We are unable to recover the namespaces. Please try again or contact support@unkey.dev",
        //       });
        //     });
        // }

        await insertAuditLogs(tx, {
          workspaceId: ws.id,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: "ratelimitNamespace.create",
          description: `Restored ${namespace.id}`,
          resources: [
            {
              type: "ratelimitNamespace",
              id: namespace.id,
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      });
      return {
        id: namespace.id,
      };
    }

    const namespaceId = newId("ratelimitNamespace");
    await db
      .transaction(async (tx) => {
        await tx.insert(schema.ratelimitNamespaces).values({
          id: namespaceId,
          name: input.name,
          workspaceId: ws.id,

          createdAt: new Date(),
        });
        await insertAuditLogs(tx, {
          workspaceId: ws.id,
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
            },
          ],
          context: {
            location: ctx.audit.location,
            userAgent: ctx.audit.userAgent,
          },
        });
      })
      .catch((e) => {
        if (e instanceof DatabaseError && e.body.message.includes("desc = Duplicate entry")) {
          throw new TRPCError({
            code: "PRECONDITION_FAILED",
            message: "duplicate namespace name. Please use a unique name for each namespace.",
          });
        }
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to create namspace. Please try again or contact support@unkey.dev",
        });
      });

    return {
      id: namespaceId,
    };
  });
