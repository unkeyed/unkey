import { db, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { ingestAuditLogs } from "@/lib/tinybird";
import { TRPCError } from "@trpc/server";
import { AesGCM, getEncryptionKeyFromEnv } from "@unkey/encryption";
import { newId } from "@unkey/id";
import { z } from "zod";
import { auth, t } from "../../trpc";

export const createGateway = t.procedure
  .use(auth)
  .input(
    z.object({
      subdomain: z.string().min(1).max(50),
      origin: z
        .string()
        .url()
        .transform((url) => url.replace("https://", "").replace("http://", "")),

      headerRewrites: z.array(
        z.object({
          name: z.string(),
          value: z.string(),
        }),
      ),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const ws = await db.query.workspaces.findFirst({
      where: (table, { and, eq, isNull }) =>
        and(eq(table.tenantId, ctx.tenant.id), isNull(table.deletedAt)),
    });
    if (!ws) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "workspace not found",
      });
    }

    const { key, version } = getEncryptionKeyFromEnv(env());

    const aes = await AesGCM.withBase64Key(key);

    const gatewayId = newId("gateway");
    await db.insert(schema.gateways).values({
      id: gatewayId,
      name: input.subdomain,
      workspaceId: ws.id,
      origin: input.origin,
    });

    const rewrites = await Promise.all(
      input.headerRewrites.map(async ({ name, value }) => {
        const secret = await aes.encrypt(value);

        return {
          secretId: newId("secret"),
          secret,
          name,
        };
      }),
    );

    if (rewrites.length > 0) {
      await db.insert(schema.secrets).values(
        rewrites.map(({ name, secretId, secret }) => ({
          id: secretId,
          algorithm: "AES-GCM" as any,
          ciphertext: secret.ciphertext,
          iv: secret.iv,
          name: `${input.subdomain}_${name}`,
          workspaceId: ws.id,
          encryptionKeyVersion: version,
          createdAt: new Date(),
        })),
      );
      await db.insert(schema.gatewayHeaderRewrites).values(
        rewrites.map(({ name, secretId }) => ({
          id: newId("headerRewrite"),
          name,
          secretId,
          createdAt: new Date(),
          workspaceId: ws.id,
          gatewayId: gatewayId,
        })),
      );
    }

    await ingestAuditLogs({
      workspaceId: ws.id,
      actor: {
        type: "user",
        id: ctx.user.id,
      },
      event: "gateway.create",
      description: `Created ${gatewayId}`,
      resources: [
        {
          type: "gateway",
          id: gatewayId,
        },
      ],
      context: {
        location: ctx.audit.location,
        userAgent: ctx.audit.userAgent,
      },
    });

    return {
      id: gatewayId,
    };
  });
