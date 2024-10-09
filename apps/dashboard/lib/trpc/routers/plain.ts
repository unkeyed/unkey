import { env } from "@/lib/env";
import { rateLimitedProcedure, ratelimit } from "@/lib/trpc/ratelimitProcedure";
import { clerkClient } from "@clerk/nextjs";
import { PlainClient, uiComponent } from "@team-plain/typescript-sdk";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

const issueType = z.enum(["bug", "feature", "security", "question", "payment"]);
const severity = z.enum(["p0", "p1", "p2", "p3"]);
export const createPlainIssue = rateLimitedProcedure(ratelimit.create)
  .input(
    z.object({
      issueType,
      severity,
      message: z.string().min(20, "Feedback must contain at least 20 characters"),
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const apiKey = env().PLAIN_API_KEY;

    if (!apiKey) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "PLAIN_API_KEY is not set",
      });
    }

    const client = new PlainClient({
      apiKey,
    });

    const user = await clerkClient.users.getUser(ctx.user.id);
    if (!user) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "User not found",
      });
    }

    const email = user.emailAddresses.at(0)!.emailAddress;
    const plainUser = await client.upsertCustomer({
      identifier: {
        emailAddress: email,
      },
      onCreate: {
        externalId: user.id,
        email: {
          email: email,
          isVerified: user.emailAddresses.at(0)?.verification?.status === "verified",
        },
        fullName: user.username ?? user.firstName ?? "none avail",
      },
      onUpdate: {
        email: {
          email: email,
          isVerified: user.emailAddresses.at(0)?.verification?.status === "verified",
        },
        fullName: { value: user.username ?? user.firstName ?? "none avail" },
      },
    });
    if (plainUser.error) {
      console.error(plainUser.error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: plainUser.error.message,
      });
    }

    const thread = await client.createThread({
      title: `${input.severity} - ${input.issueType}`,
      priority: severityToNumber[input.severity],
      customerIdentifier: {
        emailAddress: email,
      },
      components: [
        uiComponent.plainText({ text: input.message }),
        uiComponent.spacer({ spacingSize: "M" }),
        uiComponent.row({
          mainContent: [uiComponent.plainText({ text: ctx.tenant.id, color: "MUTED" })],
          asideContent: [
            uiComponent.copyButton({
              value: ctx.tenant.id,
              tooltip: "Copy Tenant Id",
            }),
          ],
        }),
      ],
      labelTypeIds: [issueToId[input.issueType]],
    });

    if (thread.error) {
      console.error(thread.error);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: thread.error.message,
      });
    }

    return;
  });
const issueToId: Record<z.infer<typeof issueType>, string> = {
  bug: "lt_01HB93K6BRT4G3SXWQAG7FS5QZ",
  feature: "lt_01HB93K6D5QQFZZ6JBZHA35NS4",
  question: "lt_01HB93K6B8Y28CBBJQV6FQ791S",
  security: "lt_01HCATRJ25F3FJ7V8W2E4CRX5H",
  payment: "lt_01HB93K6C78S0Q61VG8C3FBZZC",
};

const severityToNumber: Record<z.infer<typeof severity>, number> = {
  p0: 0,
  p1: 1,
  p2: 2,
  p3: 3,
};
