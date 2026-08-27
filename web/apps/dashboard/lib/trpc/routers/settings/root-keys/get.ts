import { and, db, eq, schema } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { rootKeyBaseConditions, rootKeyGrants } from "./shared";

const RootKeyDetailResponse = z.object({
  id: z.string(),
  start: z.string(),
  name: z.string().nullable(),
  grants: z.array(z.string()),
});

export type RootKeyDetail = z.infer<typeof RootKeyDetailResponse>;

export const getRootKey = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .input(z.object({ keyId: z.string() }))
  .output(RootKeyDetailResponse)
  .query(async ({ ctx, input }) => {
    const key = await db.query.keys
      .findFirst({
        where: and(eq(schema.keys.id, input.keyId), ...rootKeyBaseConditions(ctx.workspace.id)),
        columns: {
          id: true,
          start: true,
          name: true,
        },
        with: {
          permissions: {
            columns: {
              permissionId: true,
            },
            with: {
              permission: {
                columns: {
                  id: true,
                  name: true,
                },
              },
            },
          },
        },
      })
      .catch((error) => {
        console.error("Error reading root key:", error);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "Failed to retrieve the Root Key due to an error. If this issue persists, please contact support@unkey.com with the time this occurred.",
        });
      });

    if (!key) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "Root Key not found. It may have been deleted, expired, or belong to another workspace.",
      });
    }

    return {
      id: key.id,
      start: key.start,
      name: key.name,
      grants: rootKeyGrants(key.permissions),
    };
  });
