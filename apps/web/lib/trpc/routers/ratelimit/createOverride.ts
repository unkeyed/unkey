import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { and, db, eq, isNull, schema, sql } from "@/lib/db";
import { ingestAuditLogs } from "@/lib/tinybird";
import { newId } from "@unkey/id";
import { auth, t } from "../../trpc";

export const createOverride = t.procedure
  .use(auth)
  .input(
    z.object({
      namespaceId: z.string(),
      identifier: z.string(),
      limit: z.number(),
      duration: z.number(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const namespace = await db.query.ratelimitNamespaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.id, input.namespaceId), isNull(table.deletedAt)),
      with: {
        workspace: {
          columns: {
            id: true,
            tenantId: true,
            features: true,
          },
        },
      },
    });
    if (!namespace || namespace.workspace.tenantId !== ctx.tenant.id) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "namespace not found",
      });
    }

    const id = newId("ratelimitOverride");

    await db.transaction(async (tx) => {
      const existing = await tx
        .select({ count: sql`count(*)` })
        .from(schema.ratelimitOverrides)
        .where(
          and(
            eq(schema.ratelimitOverrides.namespaceId, namespace.id),
            isNull(schema.ratelimitOverrides.deletedAt),
          ),
        )
        .then((res) => Number(res.at(0)?.count ?? 0));
      const max =
        typeof namespace.workspace.features.ratelimitOverrides === "number"
          ? namespace.workspace.features.ratelimitOverrides
          : 5;
      if (existing >= max) {
        throw new TRPCError({
          code: "FORBIDDEN",
          message: `Upgrade required, you can only override ${max} identifiers`,
        });
      }

      await tx.insert(schema.ratelimitOverrides).values({
        workspaceId: namespace.workspace.id,
        namespaceId: namespace.id,
        identifier: input.identifier,
        id,
        limit: input.limit,
        duration: input.duration,
        createdAt: new Date(),
      });
    });

    await ingestAuditLogs({
      workspaceId: namespace.workspace.id,
      actor: {
        type: "user",
        id: ctx.user.id,
      },
      event: "ratelimitOverride.create",
      description: `Created ${input.identifier}`,
      resources: [
        {
          type: "ratelimitNamespace",
          id: input.namespaceId,
        },
        {
          type: "ratelimitOverride",
          id,
        },
      ],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });

    return {
      id,
    };
  });
