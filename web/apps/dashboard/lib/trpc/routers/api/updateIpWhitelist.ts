import { TRPCError } from "@trpc/server";
import { z } from "zod";

import { insertAuditLogs } from "@/lib/audit";
import { db, eq, schema } from "@/lib/db";

import { workspaceProcedure } from "../../trpc";

export const updateApiIpWhitelist = workspaceProcedure
  .input(
    z.object({
      ipWhitelist: z
        .string()
        .transform((s, ctx) => {
          if (s === "") {
            return null;
          }
          const ips = s.split(/,|\n/).map((ip) => ip.trim());
          const parsedIps = z.array(z.union([z.ipv4(), z.ipv6()])).safeParse(ips);
          if (!parsedIps.success) {
            const firstIssue = parsedIps.error.issues[0];
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              message: firstIssue?.message || "Invalid IP address format",
              path: firstIssue?.path || [],
            });
            return z.NEVER;
          }
          return parsedIps.data;
        })
        .nullable(),
      apiId: z.string(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const api = await db.query.apis
      .findFirst({
        where: (table, { eq, and, isNull }) =>
          and(
            eq(table.workspaceId, ctx.workspace.id),
            eq(table.id, input.apiId),
            isNull(table.deletedAtM),
          ),
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message:
            "We are unable to update the API whitelist. Please try again or contact support@unkey.dev",
        });
      });

    if (!api) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message:
          "We are unable to find the correct API. Please try again or contact support@unkey.dev.",
      });
    }

    if (!ctx.workspace.features.ipWhitelist) {
      throw new TRPCError({
        code: "FORBIDDEN",
        message:
          "IP Whitelisting is only available for enterprise plans. Please contact support@unkey.dev.",
      });
    }

    const newIpWhitelist = input.ipWhitelist === null ? null : input.ipWhitelist.join(",");

    await db
      .transaction(async (tx) => {
        await tx
          .update(schema.apis)
          .set({
            ipWhitelist: newIpWhitelist,
          })
          .where(eq(schema.apis.id, input.apiId))
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "We are unable to update the API whitelist. Please try again or contact support@unkey.dev",
            });
          });

        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          event: "api.update",
          description: `Changed ${api.id} IP whitelist from ${api.ipWhitelist} to ${newIpWhitelist}`,
          resources: [
            {
              type: "api",
              id: api.id,
              name: api.name,
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
            "We are unable to update the API whitelist. Please try again or contact support@unkey.dev",
        });
      });
  });
