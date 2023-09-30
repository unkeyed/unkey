"use server";

import { serverAction } from "@/lib/actions";
import { db, eq, schema } from "@/lib/db";
import { revalidatePath } from "next/cache";
import { z } from "zod";

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

export const updateKeyRemaining = serverAction({
  input: z.object({
    keyId: z.string(),
    enableRemaining: z.string().transform((s) => s === "true"),
    remaining: stringToIntOrNull,
  }),
  handler: async ({ input, ctx }) => {
    const key = await db().query.keys.findFirst({
      where: eq(schema.keys.id, input.keyId),
      with: {
        workspace: true,
      },
    });
    if (!key || key.workspace.tenantId !== ctx.tenantId) {
      throw new Error("key not found");
    }

    if (input.enableRemaining && typeof input.remaining !== "number") {
      throw new Error("provide a number");
    }

    await db()
      .update(schema.keys)
      .set({
        remainingRequests: input.enableRemaining ? input.remaining : null,
      })
      .where(eq(schema.keys.id, input.keyId));

    revalidatePath(`/apps/keys/${input.keyId}`);
  },
});

export const updateKeyRatelimit = serverAction({
  input: z.object({
    keyId: z.string(),
    enabled: z.string().transform((s) => s === "true"),
    ratelimitType: z.enum(["fast"]).nullable(),
    ratelimitLimit: stringToIntOrNull,
    ratelimitRefillRate: stringToIntOrNull,
    ratelimitRefillInterval: stringToIntOrNull,
  }),
  handler: async ({ input, ctx }) => {
    const key = await db().query.keys.findFirst({
      where: eq(schema.keys.id, input.keyId),
      with: {
        workspace: true,
      },
    });
    if (!key || key.workspace.tenantId !== ctx.tenantId) {
      throw new Error("key not found");
    }

    let ratelimitType: "fast" | null = null;
    let ratelimitLimit: number | null = null;
    let ratelimitRefillRate: number | null = null;
    let ratelimitRefillInterval: number | null = null;

    if (input.enabled) {
      if (typeof input.ratelimitType !== "string") {
        throw new Error("Type must be defined");
      }
      ratelimitType = input.ratelimitType;

      if (typeof input.ratelimitLimit !== "number" || input.ratelimitLimit <= 0) {
        throw new Error("Limit must be a positive integer");
      }
      ratelimitLimit = input.ratelimitLimit;

      if (typeof input.ratelimitRefillRate !== "number" || input.ratelimitRefillRate <= 0) {
        throw new Error("Rate must be a positive integer");
      }
      ratelimitRefillRate = input.ratelimitRefillRate;
      if (typeof input.ratelimitRefillInterval !== "number" || input.ratelimitRefillInterval <= 0) {
        throw new Error("Interval must be a positive integer");
      }
      ratelimitRefillInterval = input.ratelimitRefillInterval;
    }

    await db()
      .update(schema.keys)
      .set({
        ratelimitType,
        ratelimitLimit,
        ratelimitRefillRate,
        ratelimitRefillInterval,
      })
      .where(eq(schema.keys.id, input.keyId));

    revalidatePath(`/apps/keys/${input.keyId}`);
  },
});

export const updateExpiration = serverAction({
  input: z.object({
    keyId: z.string(),
    enableExpiration: z.string().transform((s) => s === "true"),
    expiration: z.string().nullish(),
  }),
  handler: async ({ input, ctx }) => {
    const key = await db().query.keys.findFirst({
      where: eq(schema.keys.id, input.keyId),
      with: {
        workspace: true,
      },
    });
    if (!key || key.workspace.tenantId !== ctx.tenantId) {
      throw new Error("key not found");
    }

    let expires: Date | null = null;
    if (input.enableExpiration) {
      if (!input.expiration) {
        throw new Error("you must enter a valid date");
      }
      try {
        expires = new Date(input.expiration);
      } catch (e) {
        console.error(e);
        throw new Error(`you must enter a valid date: ${(e as Error).message}`);
      }
    }

    await db()
      .update(schema.keys)
      .set({
        expires,
      })
      .where(eq(schema.keys.id, input.keyId));

    revalidatePath(`/apps/keys/${input.keyId}`);
  },
});

export const updateMetadata = serverAction({
  input: z.object({
    keyId: z.string(),
    metadata: z.string().nullish(),
  }),
  handler: async ({ input, ctx }) => {
    const key = await db().query.keys.findFirst({
      where: eq(schema.keys.id, input.keyId),
      with: {
        workspace: true,
      },
    });
    if (!key || key.workspace.tenantId !== ctx.tenantId) {
      throw new Error("key not found");
    }

    await db()
      .update(schema.keys)
      .set({
        meta: input.metadata ?? null,
      })
      .where(eq(schema.keys.id, input.keyId));

    revalidatePath(`/apps/keys/${input.keyId}`);
  },
});

export const updateKeyName = serverAction({
  input: z.object({
    keyId: z.string(),
    name: z.string().nullish(),
  }),
  handler: async ({ input, ctx }) => {
    const key = await db().query.keys.findFirst({
      where: eq(schema.keys.id, input.keyId),
      with: {
        workspace: true,
      },
    });
    if (!key || key.workspace.tenantId !== ctx.tenantId) {
      throw new Error("key not found");
    }

    await db()
      .update(schema.keys)
      .set({
        name: input.name ?? null,
      })
      .where(eq(schema.keys.id, input.keyId));

    revalidatePath(`/apps/keys/${input.keyId}`);
  },
});
