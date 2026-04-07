import { db } from "@/lib/db";
import { getStripeClient } from "@/lib/stripe";
import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { z } from "zod";
import { protectedProcedure } from "../../trpc";

export const getWorkspaceById = protectedProcedure
  .input(
    z.object({
      workspaceId: z.string().min(1, "Workspace ID is required"),
      sessionId: z.string().min(1, "Session ID is required"),
    }),
  )
  .query(async ({ input }) => {
    try {
      const stripe = getStripeClient();
      const session = await stripe.checkout.sessions.retrieve(input.sessionId);

      if (!session || session.client_reference_id !== input.workspaceId) {
        throw new TRPCError({
          code: "FORBIDDEN",
          message: "Invalid Stripe session for this workspace",
        });
      }

      const workspace = await db.query.workspaces.findFirst({
        where: (table, { eq, and, isNull }) =>
          and(eq(table.id, input.workspaceId), isNull(table.deletedAtM)),
        columns: {
          slug: true,
        },
      });

      if (!workspace) {
        throw new TRPCError({
          code: "NOT_FOUND",
          message: "Workspace not found",
        });
      }

      return { slug: workspace.slug };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }

      if (error instanceof Stripe.errors.StripeError) {
        throw new TRPCError({
          code: "FORBIDDEN",
          message: "Invalid Stripe session",
        });
      }

      console.error("Error fetching workspace by ID:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch workspace data",
        cause: error,
      });
    }
  });
