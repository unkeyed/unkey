import { insertAuditLogs } from "@/lib/audit";
import { db, schema } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { newId } from "@unkey/id";
import { z } from "zod";
import { workspaceProcedure } from "../../trpc";
const nameSchema = z
  .string()
  .min(3)
  .regex(/^[a-zA-Z0-9_:\-\.\*]+$/, {
    error:
      "Must be at least 3 characters long and only contain alphanumeric, colons, periods, dashes and underscores",
  });

export const createRole = workspaceProcedure
  .input(
    z.object({
      name: nameSchema,
      description: z.string().optional(),
      permissionIds: z.array(z.string()).optional(),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const roleId = newId("role");
    await db
      .transaction(async (tx) => {
        const existing = await tx.query.roles.findFirst({
          where: (table, { and, eq }) =>
            and(eq(table.workspaceId, ctx.workspace.id), eq(table.name, input.name)),
        });
        if (existing) {
          throw new TRPCError({
            code: "CONFLICT",
            message:
              "Role with the same name already exists. To update it, go to 'Authorization' in the sidebar.",
          });
        }

        await tx
          .insert(schema.roles)
          .values({
            id: roleId,
            name: input.name,
            description: input.description,
            workspaceId: ctx.workspace.id,
          })
          .catch((_err) => {
            throw new TRPCError({
              code: "INTERNAL_SERVER_ERROR",
              message:
                "We are unable to create a role. Please try again or contact support@unkey.dev.",
            });
          });
        await insertAuditLogs(tx, {
          workspaceId: ctx.workspace.id,
          event: "role.create",
          actor: {
            type: "user",
            id: ctx.user.id,
          },
          description: `Created ${roleId}`,
          resources: [
            {
              type: "role",
              id: roleId,
              name: input.name,
            },
          ],

          context: {
            userAgent: ctx.audit.userAgent,
            location: ctx.audit.location,
          },
        });

        if (input.permissionIds && input.permissionIds.length > 0) {
          await tx.insert(schema.rolesPermissions).values(
            input.permissionIds.map((permissionId) => ({
              permissionId,
              roleId: roleId,
              workspaceId: ctx.workspace.id,
            })),
          );
          await insertAuditLogs(
            tx,

            input.permissionIds.map((permissionId) => ({
              workspaceId: ctx.workspace.id,
              event: "authorization.connect_role_and_permission",
              actor: {
                type: "user",
                id: ctx.user.id,
              },
              description: `Connected ${roleId} and ${permissionId}`,
              resources: [
                { type: "role", id: roleId, name: input.name },
                {
                  type: "permission",
                  id: permissionId,
                },
              ],

              context: {
                userAgent: ctx.audit.userAgent,
                location: ctx.audit.location,
              },
            })),
          );
        }
      })
      .catch((_err) => {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "We are unable to create role. Please try again or contact support@unkey.dev",
        });
      });
    return { roleId };
  });
