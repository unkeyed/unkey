import { createKeyInputSchema } from "@/app/(app)/apis/[apiId]/_components/create-key/create-key.schema";
import { insertAuditLogs } from "@/lib/audit";
import { db, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { newKey } from "@unkey/keys";
import { requireUser, requireWorkspace, t } from "../../trpc";

export const createKey = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(createKeyInputSchema)
  .mutation(async ({ input, ctx }) => {
    const keyAuth = await db.query.keyAuth
      .findFirst({
        where: (table, { and, eq }) =>
          and(eq(table.workspaceId, ctx.workspace.id), eq(table.id, input.keyAuthId)),
        with: {
          api: true,
        },
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to create a key for this API. Please try again or contact support@unkey.dev.",
        });
      });
    if (!keyAuth) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct keyAuth. Please try again or contact support@unkey.dev",
      });
    }

    const keyId = newId("key");
    const { key, hash, start } = await newKey({
      prefix: input.prefix,
      byteLength: input.bytes,
    });
    await db
      .transaction(async (tx) => {
        await tx.insert(schema.keys).values({
          id: keyId,
          keyAuthId: keyAuth.id,
          name: input.name,
          hash,
          start,
          ownerId: input.ownerId,
          meta: JSON.stringify(input.meta ?? {}),
          workspaceId: ctx.workspace.id,
          forWorkspaceId: null,
          expires: input.expires ? new Date(input.expires) : null,
          createdAtM: Date.now(),
          updatedAtM: null,
          remaining: input.remaining,
          refillDay: input.refill?.refillDay ?? null,
          refillAmount: input.refill?.amount ?? null,
          lastRefillAt: input.refill ? new Date() : null,
          enabled: input.enabled,
          environment: input.environment,
        });

        if (input.ratelimit?.length) {
          await tx.insert(schema.ratelimits).values(
            input.ratelimit.map((ratelimit) => ({
              id: newId("ratelimit"),
              keyId,
              duration: ratelimit.refillInterval,
              limit: ratelimit.limit,
              name: ratelimit.name,
              workspaceId: ctx.workspace.id,
              createdAt: Date.now(),
              updatedAt: null,
            })),
          );
        }

        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: { type: "user", id: ctx.user.id },
          event: "key.create",
          description: `Created ${keyId}`,
          resources: [
            {
              type: "key",
              id: keyId,
              name: input.name,
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
            "We are unable to create the key. Please contact support using support.unkey.dev",
        });
      });

    return { keyId, key, name: input.name };
  });
