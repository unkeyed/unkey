import { env } from "@/lib/env";
import { TRPCError } from "@trpc/server";
import { Resend } from "resend";
import { z } from "zod";
import { t } from "../trpc";

export const newsletterRouter = t.router({
  signup: t.procedure
    .input(
      z.object({
        email: z.string(),
      }),
    )
    .mutation(async ({ input }) => {
      const { RESEND_API_KEY, RESEND_AUDIENCE_ID } = env();

      if (!RESEND_API_KEY || !RESEND_AUDIENCE_ID) {
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: "An unexpected error occurred, please try again later.",
        });
      }

      const email = input.email;
      if (!email) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "Email is not valid. Please try again.",
        });
      }
      const resend = new Resend(RESEND_API_KEY);
      const res = await resend.contacts.create({
        audience_id: RESEND_AUDIENCE_ID,
        email: email,
      });
      if (res.error) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "Resend Error. Please try again.",
        });
      }
      return res.data?.id ? true : false;
    }),
});
