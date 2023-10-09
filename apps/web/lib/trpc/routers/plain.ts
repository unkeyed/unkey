import { env } from "@/lib/env";
import { clerkClient } from "@clerk/nextjs";
import { PlainClient } from "@team-plain/typescript-sdk";
import { ComponentSpacerSize } from "@team-plain/typescript-sdk";
import { ComponentTextSize } from "@team-plain/typescript-sdk";
import { ComponentTextColor } from "@team-plain/typescript-sdk";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { auth, t } from "../trpc";

const issueType = z.enum(["bug", "feature", "security", "question", "payment"]);
const severity = z.enum(["p0", "p1", "p2", "p3"]);
export const plainRouter = t.router({
  createIssue: t.procedure
    .use(auth)
    .input(
      z.object({
        issueType,
        severity,
        message: z.string(),
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

      console.log({ user });

      const plainUser = await client.upsertCustomer({
        identifier: {
          externalId: user.id,
        },
        onCreate: {
          email: {
            email: user.emailAddresses.at(0)?.emailAddress ?? "",
            isVerified: user.emailAddresses.at(0)?.verification?.status === "verified",
          },
          fullName: user.username ?? "",
        },
        onUpdate: {},
      });
      if (plainUser.error) {
        console.error(plainUser.error);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: plainUser.error.message,
        });
      }

      console.log({ plainUser });
      const thread = await client.createThread({
        title: input.message,
        priority: severityToNumber[input.severity],
        customerIdentifier: {
          externalId: user.id,
        },
        components: [],
        labelTypeIds: [issueToId[input.issueType]],
      });

      if (thread.error) {
        console.error(thread.error.message);
        throw new TRPCError({
          code: "INTERNAL_SERVER_ERROR",
          message: thread.error.message,
        });
      }

      return;
    }),
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

export function customTimelineEntryForBug(text: string, path: string | null) {
  return {
    title: "Bug report",
    components: [
      {
        componentText: {
          text,
        },
      },
      {
        componentSpacer: {
          spacerSize: ComponentSpacerSize.S,
        },
      },
      {
        componentText: {
          text: `Reported on ${path}`,
          textSize: ComponentTextSize.S,
          textColor: ComponentTextColor.Muted,
        },
      },
    ],
  };
}

function customTimelineEntryForFeatureRequest(text: string) {
  return {
    title: "Feature request",
    components: [
      {
        componentText: {
          text,
        },
      },
    ],
  };
}

function customTimelineEntryForQuestion(text: string) {
  return {
    title: "General question",
    components: [
      {
        componentText: {
          text,
        },
      },
    ],
  };
}

function customTimelineEntryForSecurityReport(text: string) {
  return {
    title: "Security report",
    components: [
      {
        componentText: {
          text,
        },
      },
    ],
  };
}
