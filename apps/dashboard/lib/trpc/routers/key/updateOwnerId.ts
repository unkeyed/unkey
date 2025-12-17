import { type UnkeyAuditLog, insertAuditLogs } from "@/lib/audit";
import { type Key, and, db, eq, inArray, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";

const baseOwnerInputSchema = z.object({
  keyIds: z
    .union([z.string(), z.array(z.string())])
    .transform((ids) => (Array.isArray(ids) ? ids : [ids])),
});

const ownerValidationV1 = z.object({
  ownerId: z.string().nullish(),
  ownerType: z.literal("v1"),
});

const ownerValidationV2 = z.object({
  identity: z.object({
    id: z.string().nullish(),
    externalId: z.string().nullish(),
  }),
  ownerType: z.literal("v2"),
});

export const ownerInputSchema = z
  .discriminatedUnion("ownerType", [ownerValidationV1, ownerValidationV2])
  .and(baseOwnerInputSchema);

type OwnerInputSchema = z.infer<typeof ownerInputSchema>;

export const updateKeyOwner = workspaceProcedure
  .input(ownerInputSchema)
  .mutation(async ({ input, ctx }) => {
    // Ensure we have at least one keyId to update
    if (input.keyIds.length === 0) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: "At least one keyId must be provided",
      });
    }

    // Find all keys that match the criteria
    const keys = await db.query.keys
      .findMany({
        where: (table, { eq, and, isNull, inArray }) =>
          and(
            eq(table.workspaceId, ctx.workspace.id),
            inArray(table.id, input.keyIds),
            isNull(table.deletedAtM),
          ),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We were unable to retrieve keys to update owner information. Please try again or contact support@unkey.dev",
        });
      });

    // Check if we found all the requested keys
    if (keys.length === 0) {
      throw new TRPCError({
        message:
          "We were unable to find any of the specified keys. Please try again or contact support@unkey.dev.",
        code: "NOT_FOUND",
      });
    }

    // Warn if some keys weren't found
    const foundKeyIds = new Set(keys.map((key) => key.id));
    const missingKeyIds = input.keyIds.filter((id) => !foundKeyIds.has(id));

    if (missingKeyIds.length > 0) {
      console.warn(`Some keys were not found: ${missingKeyIds.join(", ")}`);
    }

    if (input.ownerType === "v1") {
      return updateOwnerV1(input, keys, {
        audit: ctx.audit,
        userId: ctx.user.id,
        workspaceId: ctx.workspace.id,
      });
    }
    return updateOwnerV2(input, keys, {
      audit: ctx.audit,
      userId: ctx.user.id,
      workspaceId: ctx.workspace.id,
    });
  });

const updateOwnerV1 = async (
  input: OwnerInputSchema,
  keys: Key[],
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
      // Update all keys in a single query
      await tx
        .update(schema.keys)
        .set({
          ownerId: input.ownerId ?? null,
        })
        .where(
          and(
            eq(schema.keys.workspaceId, ctx.workspaceId),
            inArray(
              schema.keys.id,
              keys.map((key) => key.id),
            ),
          ),
        );

      const keyIds = keys.map((key) => key.id).join(", ");
      const description = `Changed ownerId of keys [${keyIds}] to ${input.ownerId ?? "null"}`;

      const resources: UnkeyAuditLog["resources"] = keys.map((key) => ({
        type: "key",
        id: key.id,
        name: key.name || undefined,
        meta: {
          "owner.id": input.ownerId ?? null,
        },
      }));

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
        "We were unable to update owner information on these keys. Please try again or contact support@unkey.dev",
    });
  }

  return {
    keyIds: keys.map((key) => key.id),
    updatedCount: keys.length,
  };
};

const updateOwnerV2 = async (
  input: OwnerInputSchema,
  keys: Key[],
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
          ownerId: input.identity.externalId ?? null,
        })
        .where(
          and(
            eq(schema.keys.workspaceId, ctx.workspaceId),
            inArray(
              schema.keys.id,
              keys.map((key) => key.id),
            ),
          ),
        );

      const keyIds = keys.map((key) => key.id).join(", ");
      const description = `Updated identity of keys [${keyIds}] to ${input.identity.id ?? "null"}`;

      const resources: UnkeyAuditLog["resources"] = keys.map((key) => ({
        type: "key",
        id: key.id,
        name: key.name || undefined,
        meta: {
          "identity.id": input.identity.id ?? null,
        },
      }));

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
        "We were unable to update identity information on these keys. Please try again or contact support@unkey.dev",
    });
  }

  return {
    keyIds: keys.map((key) => key.id),
    updatedCount: keys.length,
  };
};
