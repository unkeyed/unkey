import {
  type CreateKeyInput,
  createKeyInputSchema,
} from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/create-key.schema";
import { insertAuditLogs } from "@/lib/audit";
import { db, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { Vault } from "@/lib/vault";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { newKey } from "@unkey/keys";
import { ratelimit, withRatelimit, workspaceProcedure } from "../../trpc";

const vault = new Vault({
  baseUrl: env().AGENT_URL,
  token: env().AGENT_TOKEN,
});

export const createKey = workspaceProcedure
  .use(withRatelimit(ratelimit.create))
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
            "We were unable to create a key for this API. Please try again or contact support@unkey.com.",
        });
      });

    if (!keyAuth) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct keyAuth. Please try again or contact support@unkey.com",
      });
    }

    try {
      return await db.transaction(async (tx) => {
        return await createKeyCore(
          {
            ...input,
            keyAuthId: keyAuth.id,
            storeEncryptedKeys: keyAuth.storeEncryptedKeys,
          },
          ctx,
          tx,
        );
      });
    } catch (_err) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "We are unable to create the key. Please contact support using support.unkey.dev",
      });
    }
  });

type CreateKeyContext = {
  workspace: { id: string };
  user: { id: string };
  audit: {
    location: string;
    userAgent?: string;
  };
};

type DatabaseTransaction = Parameters<Parameters<typeof db.transaction>[0]>[0];

export async function createKeyCore(
  input: CreateKeyInput & { storeEncryptedKeys: boolean },
  ctx: CreateKeyContext,
  tx: DatabaseTransaction,
) {
  const keyId = newId("key");
  const { key, hash, start } = await newKey({
    prefix: input.prefix,
    byteLength: input.bytes,
  });

  await tx.insert(schema.keys).values({
    id: keyId,
    keyAuthId: input.keyAuthId,
    name: input.name,
    hash,
    start,
    identityId: input.identityId,
    ownerId: input.externalId,
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

  if (input.storeEncryptedKeys) {
    const { encrypted, keyId: encryptionKeyId } = await vault.encrypt({
      keyring: ctx.workspace.id,
      data: key,
    });

    await tx.insert(schema.encryptedKeys).values({
      encrypted,
      encryptionKeyId,
      keyId,
      workspaceId: ctx.workspace.id,
      createdAt: Date.now(),
      updatedAt: null,
    });
  }

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
        autoApply: ratelimit.autoApply,
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

  return { keyId, key, name: input.name };
}
