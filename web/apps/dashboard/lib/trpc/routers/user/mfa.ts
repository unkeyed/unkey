import { auth as authProvider } from "@/lib/auth/server";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { protectedProcedure } from "../../trpc";

export const listMfaFactors = protectedProcedure.query(async ({ ctx }) => {
  try {
    return await authProvider.listMfaFactors(ctx.user.id);
  } catch (error) {
    console.error("Error listing MFA factors:", error);
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Failed to list MFA factors",
      cause: error,
    });
  }
});

export const startMfaEnrollment = protectedProcedure.mutation(async ({ ctx }) => {
  try {
    const user = await authProvider.getUser(ctx.user.id);
    if (!user) {
      throw new TRPCError({ code: "NOT_FOUND", message: "User not found" });
    }

    return await authProvider.beginMfaEnrollment({
      userId: ctx.user.id,
      email: user.email,
    });
  } catch (error) {
    if (error instanceof TRPCError) {
      throw error;
    }
    console.error("Error starting MFA enrollment:", error);
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Failed to start MFA enrollment",
      cause: error,
    });
  }
});

export const verifyMfaEnrollment = protectedProcedure
  .input(
    z.object({
      challengeId: z.string().min(1),
      code: z.string().length(6),
    }),
  )
  .mutation(async ({ input }) => {
    try {
      const valid = await authProvider.verifyMfaEnrollment(input);
      return { valid };
    } catch (error) {
      console.error("Error verifying MFA enrollment:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to verify the code",
        cause: error,
      });
    }
  });

export const removeMfaFactor = protectedProcedure
  .input(z.object({ factorId: z.string().min(1) }))
  .mutation(async ({ ctx, input }) => {
    try {
      // Only allow deleting factors that belong to the caller
      const factors = await authProvider.listMfaFactors(ctx.user.id);
      if (!factors.some((factor) => factor.id === input.factorId)) {
        throw new TRPCError({ code: "NOT_FOUND", message: "Factor not found" });
      }

      await authProvider.removeMfaFactor(input.factorId);
      return { success: true };
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      console.error("Error removing MFA factor:", error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to remove MFA factor",
        cause: error,
      });
    }
  });
