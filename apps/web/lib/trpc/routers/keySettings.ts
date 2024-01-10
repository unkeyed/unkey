import { db, eq, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { auth, t } from "../trpc";

const stringToIntOrNull = z
  .string()
  .nullish()
  .transform((s, ctx) => {
    if (!s || s === "") {
      return null;
    }

    const parsed = z.number().nonnegative().int().nullable().safeParse(parseInt(s));
    if (!parsed.success) {
      ctx.addIssue(parsed.error.issues[0]);
      return z.NEVER;
    }
    return parsed.data;
  });
export const keySettingsRouter = t.router({
  updateRatelimit: t.procedure
    .use(auth)
    .input(
      z.object({
        keyId: z.string(),
        enabled: z.string().transform((s) => s === "true"),
        ratelimitType: z.enum(["fast"]).nullable(),
        ratelimitLimit: stringToIntOrNull,
        ratelimitRefillRate: stringToIntOrNull,
        ratelimitRefillInterval: stringToIntOrNull,
      }),
    )
    .mutation(async ({ input, ctx }) => {
      let ratelimitType: "fast" | null = null;
      let ratelimitLimit: number | null = null;
      let ratelimitRefillRate: number | null = null;
      let ratelimitRefillInterval: number | null = null;

      if (input.enabled) {
        if (typeof input.ratelimitType !== "string") {
          throw new TRPCError({ message: "ratelimitType must be a string", code: "BAD_REQUEST" });
        }
        ratelimitType = input.ratelimitType;

        if (typeof input.ratelimitLimit !== "number" || input.ratelimitLimit <= 0) {
          throw new TRPCError({ message: "Limit must be a positive integer", code: "BAD_REQUEST" });
        }
        ratelimitLimit = input.ratelimitLimit;

        if (typeof input.ratelimitRefillRate !== "number" || input.ratelimitRefillRate <= 0) {
          throw new TRPCError({ message: "Rate must be a positive integer", code: "BAD_REQUEST" });
        }
        ratelimitRefillRate = input.ratelimitRefillRate;
        if (
          typeof input.ratelimitRefillInterval !== "number" ||
          input.ratelimitRefillInterval <= 0
        ) {
          throw new TRPCError({
            message: "Interval must be a positive integer",
            code: "BAD_REQUEST",
          });
        }
        ratelimitRefillInterval = input.ratelimitRefillInterval;
      }
      const key = await db.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, input.keyId),
        with: {
          workspace: true,
        },
      });
      if (!key) {
        throw new TRPCError({ message: "key not found", code: "NOT_FOUND" });
      }
      if (key.workspace.tenantId !== ctx.tenant.id) {
        throw new TRPCError({ message: "key not found", code: "NOT_FOUND" });
      }
      const _result = await db.transaction(async (tx) => {
        await tx
          .update(schema.keys)
          .set({
            ratelimitType,
            ratelimitLimit,
            ratelimitRefillRate,
            ratelimitRefillInterval,
          })
          .where(eq(schema.keys.id, input.keyId));
      });
    }),
  updateOwnerId: t.procedure
    .use(auth)
    .input(
      z.object({
        keyId: z.string(),
        ownerId: z.string().nullish(),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      const key = await db.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, input.keyId),
        with: {
          workspace: true,
        },
      });
      if (!key) {
        throw new TRPCError({ message: "key not found", code: "NOT_FOUND" });
      }
      if (key.workspace.tenantId !== ctx.tenant.id) {
        throw new TRPCError({ message: "key not found", code: "NOT_FOUND" });
      }
      await db.transaction(async (tx) => {
        await tx
          .update(schema.keys)
          .set({
            ownerId: input.ownerId ?? null,
          })
          .where(eq(schema.keys.id, input.keyId));
      });

      return true;
    }),
  updateName: t.procedure
    .use(auth)
    .input(
      z.object({
        keyId: z.string(),
        name: z.string().nullish(),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      const key = await db.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, input.keyId),
        with: {
          workspace: true,
        },
      });
      if (!key) {
        throw new TRPCError({ message: "key not found", code: "NOT_FOUND" });
      }
      if (key.workspace.tenantId !== ctx.tenant.id) {
        throw new TRPCError({ message: "key not found", code: "NOT_FOUND" });
      }
      await db.transaction(async (tx) => {
        await tx
          .update(schema.keys)
          .set({
            name: input.name ?? null,
          })
          .where(eq(schema.keys.id, input.keyId));
      });
      return true;
    }),
  updateMetadata: t.procedure
    .use(auth)
    .input(
      z.object({
        keyId: z.string(),
        metadata: z.string().nullable(),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      let meta: unknown | null = null;

      if (input.metadata !== null) {
        try {
          meta = JSON.parse(input.metadata);
        } catch (e) {
          throw new TRPCError({
            message: `Metadata is not valid ${(e as Error).message}`,
            code: "BAD_REQUEST",
          });
        }
      }

      const key = await db.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, input.keyId),
        with: {
          workspace: true,
        },
      });
      if (!key) {
        throw new TRPCError({ message: "key not found", code: "NOT_FOUND" });
      }
      if (key.workspace.tenantId !== ctx.tenant.id) {
        throw new TRPCError({
          message: "key not found",
          code: "NOT_FOUND",
        });
      }
      await db.transaction(async (tx) => {
        await tx
          .update(schema.keys)
          .set({
            meta: meta ? JSON.stringify(meta) : null,
          })
          .where(eq(schema.keys.id, input.keyId));
      });
      return true;
    }),
  updateExpiration: t.procedure
    .use(auth)
    .input(
      z.object({
        keyId: z.string(),
        enableExpiration: z.boolean(),
        expiration: z.string().nullish(),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      let expires: Date | null = null;
      if (input.enableExpiration) {
        if (!input.expiration) {
          throw new TRPCError({ message: "you must enter a valid date", code: "BAD_REQUEST" });
        }
        try {
          expires = new Date(input.expiration);
        } catch (e) {
          console.error(e);
          throw new TRPCError({
            message: `you must enter a valid ${(e as Error).message}`,
            code: "BAD_REQUEST",
          });
        }
      }

      const key = await db.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, input.keyId),
        with: {
          workspace: true,
        },
      });
      if (!key) {
        throw new TRPCError({ message: "key not found", code: "NOT_FOUND" });
      }
      if (key.workspace.tenantId !== ctx.tenant.id) {
        throw new TRPCError({ message: "key not found", code: "NOT_FOUND" });
      }
      await db.transaction(async (tx) => {
        await tx
          .update(schema.keys)
          .set({
            expires,
          })
          .where(eq(schema.keys.id, input.keyId));
      });
      return true;
    }),
  updateRemaining: t.procedure
    .use(auth)
    .input(
      z.object({
        keyId: z.string(),
        enableRemaining: z.boolean().transform((s) => s === true),
        remaining: z.number().int().positive().optional(),
        refillInterval: z.enum(["null", "daily", "monthly"]).optional(),
        refillAmount: z.number().int().positive().optional(),
      }),
    )
    .mutation(async ({ input, ctx }) => {
      if (input.enableRemaining && typeof input.remaining !== "number") {
        throw new TRPCError({ message: "provide a number", code: "BAD_REQUEST" });
      }

      const key = await db.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, input.keyId),
        with: {
          workspace: true,
        },
      });
      if (!key) {
        throw new TRPCError({ message: "key not found", code: "NOT_FOUND" });
      }
      if (key.workspace.tenantId !== ctx.tenant.id) {
        throw new TRPCError({ message: "key not found", code: "NOT_FOUND" });
      }

      if (input?.enableRemaining === false || input?.remaining === null) {
        input.refillInterval = "null";
      }
      await db.transaction(async (tx) => {
        await tx
          .update(schema.keys)
          .set({
            remaining: input.enableRemaining ? input.remaining : null,
            refillInterval: input?.refillInterval !== "null" ? input.refillInterval : null,
            refillAmount: input?.refillInterval !== "null" ? input?.refillAmount : null,
          })
          .where(eq(schema.keys.id, input.keyId));
      });
      return true;
      // revalidatePath(`/apps/keys/${input.keyId}`);
    }),
});
