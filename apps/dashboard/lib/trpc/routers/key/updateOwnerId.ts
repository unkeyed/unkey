import { type UnkeyAuditLog, insertAuditLogs } from "@/lib/audit";
import { type Key, db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { requireUser, requireWorkspace, t } from "../../trpc";

const baseOwnerInputSchema = z.object({
  keyId: z.string(),
});

const ownerValidationV1 = z.object({
  ownerId: z.string().nullish(),
  ownerType: z.literal("v1"),
});

const ownerValidationV2 = z.object({
  identity: z.object({
    id: z.string().nullish(),
  }),
  ownerType: z.literal("v2"),
});

export const ownerInputSchema = z
  .discriminatedUnion("ownerType", [ownerValidationV1, ownerValidationV2])
  .and(baseOwnerInputSchema);

type OwnerInputSchema = z.infer<typeof ownerInputSchema>;

export const updateKeyOwner = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .input(ownerInputSchema)
  .mutation(async ({ input, ctx }) => {
    const key = await db.query.keys
      .findFirst({
        where: (table, { eq, and, isNull }) =>
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
            "We were unable to update owner information on this key. Please try again or contact support@unkey.dev",
        });
      });
    if (!key) {
      throw new TRPCError({
        message:
          "We are unable to find the correct key. Please try again or contact support@unkey.dev.",
        code: "NOT_FOUND",
      });
    }

    if (input.ownerType === "v1") {
      return updateOwnerV1(input, key, {
        audit: ctx.audit,
        userId: ctx.user.id,
        workspaceId: ctx.workspace.id,
      });
    }
    return updateOwnerV2(input, key, {
      audit: ctx.audit,
      userId: ctx.user.id,
      workspaceId: ctx.workspace.id,
    });
  });

const updateOwnerV1 = async (
  input: OwnerInputSchema,
  key: Key,
  ctx: {
    workspaceId: string;
    userId: string;
    audit: {
      location: string;
      userAgent: string | undefined;
    };
  },
) => {
  if (input.ownerType !== "v1") {
    throw new TRPCError({
      message: "Unsupported owner type. Only v1 owner types are supported.",
      code: "BAD_REQUEST",
    });
  }
  try {
    await db.transaction(async (tx) => {
      await tx
        .update(schema.keys)
        .set({
          ownerId: input.ownerId ?? null,
        })
        .where(eq(schema.keys.id, key.id));

      const description = `Changed ownerId of ${key.id} to ${input.ownerId ?? "null"}`;

      const resources: UnkeyAuditLog["resources"] = [
        {
          type: "key",
          id: key.id,
          name: key.name || undefined,
          meta: {
            "owner.id": input.ownerId ?? null,
          },
        },
      ];

      await insertAuditLogs(tx, {
        workspaceId: ctx.workspaceId,
        actor: {
          type: "user",
          id: ctx.userId,
        },
        event: "key.update",
        description,
        resources,
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      });
    });
  } catch (err) {
    console.error(err);
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message:
        "We were unable to update owner information on this key. Please try again or contact support@unkey.dev",
    });
  }

  return { keyId: key.id };
};

const updateOwnerV2 = async (
  input: OwnerInputSchema,
  key: Key,
  ctx: {
    workspaceId: string;
    userId: string;
    audit: {
      location: string;
      userAgent: string | undefined;
    };
  },
) => {
  if (input.ownerType !== "v2") {
    throw new TRPCError({
      message: "Unsupported owner type. Only v2 owner types are supported.",
      code: "BAD_REQUEST",
    });
  }
  try {
    await db.transaction(async (tx) => {
      await tx
        .update(schema.keys)
        .set({
          identityId: input.identity.id ?? null,
          // Set ownerId to null to maintain consistency
          ownerId: null,
        })
        .where(eq(schema.keys.id, key.id));

      const description = `Updated identity of ${key.id} to ${input.identity.id ?? "null"}`;

      const resources: UnkeyAuditLog["resources"] = [
        {
          type: "key",
          id: key.id,
          name: key.name || undefined,
          meta: {
            "identity.id": input.identity.id ?? null,
          },
        },
      ];

      await insertAuditLogs(tx, {
        workspaceId: ctx.workspaceId,
        actor: {
          type: "user",
          id: ctx.userId,
        },
        event: "key.update",
        description,
        resources,
        context: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
      });
    });
  } catch (err) {
    console.error(err);
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message:
        "We were unable to update identity information on this key. Please try again or contact support@unkey.dev",
    });
  }

  return { keyId: key.id };
};
