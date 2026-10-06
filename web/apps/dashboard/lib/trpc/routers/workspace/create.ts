import { auth as authProvider } from "@/lib/auth/server";
import { env } from "@/lib/env";
import { WorkspaceCreateError, createFreeWorkspace } from "@/lib/workspace/create-workspace";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { protectedProcedure } from "../../trpc";

export const createWorkspace = protectedProcedure
  .input(
    z.object({
      name: z.string().min(3).max(50),
      slug: z.string().regex(/^(?!-)[a-z0-9]+(?:-[a-z0-9]+)*(?<!-)$/, {
        message: "Use lowercase letters, numbers, and hyphens (no leading/trailing hyphens).",
      }),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const userId = ctx.user?.id;

    if (!userId) {
      throw new TRPCError({
        code: "UNAUTHORIZED",
        message:
          "We are not able to authenticate the user. Please make sure you are logged in and try again",
      });
    }

    try {
      const created = await createFreeWorkspace({
        name: input.name,
        slug: input.slug,
        userId,
        audit: {
          location: ctx.audit.location,
          userAgent: ctx.audit.userAgent,
        },
        createTenant: (params) => authProvider.createTenant(params),
        deleteTenant: (orgId) => authProvider.deleteTenant(orgId),
        localOrgId: env().AUTH_PROVIDER === "local" ? ctx.tenant.id : null,
      });
      return {
        orgId: created.orgId,
        slug: created.slug,
      };
    } catch (err) {
      if (err instanceof WorkspaceCreateError) {
        throw new TRPCError({
          code: err.code,
          message: err.message,
        });
      }
      if (err instanceof TRPCError) {
        throw err;
      }
      console.error(err);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message:
          "We are unable to create the workspace. Please try again or contact support@unkey.com",
      });
    }
  });
