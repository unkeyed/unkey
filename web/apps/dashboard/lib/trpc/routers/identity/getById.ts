import { db } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { ratelimit, withRatelimit, workspaceProcedure } from "../../trpc";

const GetIdentityByIdInput = z.object({
  identityId: z.string(),
});

const KeySchema = z.object({
  id: z.string(),
  meta: z.string().nullable(),
  keyAuth: z.object({
    id: z.string(),
    api: z.object({
      id: z.string(),
    }),
  }),
});

const RatelimitSchema = z.object({
  id: z.string(),
  name: z.string(),
  limit: z.number(),
  duration: z.number(),
});

const WorkspaceSchema = z.object({
  id: z.string(),
  orgId: z.string(),
  slug: z.string(),
});

const IdentityDetailSchema = z.object({
  id: z.string(),
  externalId: z.string(),
  meta: z.record(z.string(), z.unknown()).nullable(),
  workspace: WorkspaceSchema,
  keys: z.array(KeySchema),
  ratelimits: z.array(RatelimitSchema),
});

export const getIdentityById = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(GetIdentityByIdInput)
  .output(IdentityDetailSchema)
  .query(async ({ ctx, input }) => {
    const { identityId } = input;
    const { workspace } = ctx;

    try {
      const identity = await db.query.identities.findFirst({
        where: (table, { eq }) => eq(table.id, identityId),
        with: {
          workspace: {
            columns: {
              id: true,
              orgId: true,
              slug: true,
            },
          },
          keys: {
            where: (table, { isNull }) => isNull(table.deletedAtM),
            with: {
              keyAuth: {
                with: {
                  api: true,
                },
              },
            },
          },
          ratelimits: true,
        },
      });

      if (!identity || identity.workspace.orgId !== workspace.orgId) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Identity not found",
        });
      }

      return {
        id: identity.id,
        externalId: identity.externalId,
        meta: identity.meta,
        workspace: {
          id: identity.workspace.id,
          orgId: identity.workspace.orgId,
          slug: identity.workspace.slug,
        },
        keys: identity.keys.map((key) => ({
          id: key.id,
          meta: key.meta,
          keyAuth: {
            id: key.keyAuth.id,
            api: {
              id: key.keyAuth.api.id,
            },
          },
        })),
        ratelimits: identity.ratelimits.map((ratelimit) => ({
          id: ratelimit.id,
          name: ratelimit.name,
          limit: ratelimit.limit,
          duration: ratelimit.duration,
        })),
      };
    } catch (error) {
      console.error("Error retrieving identity:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "Failed to retrieve identity. If this issue persists, please contact support@unkey.com with the time this occurred.",
      });
    }
  });
