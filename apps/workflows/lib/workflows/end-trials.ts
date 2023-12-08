import { connectDatabase, eq, schema } from "@/lib/db";
import { inngest } from "@/lib/inngest";
import { clerkClient } from "@clerk/nextjs";
import { Resend } from "@unkey/resend";
import { env } from "../env";

export const endTrials = inngest.createFunction(
  {
    id: "billing/end.trials",
  },
  { cron: "0 * * * *" }, // every hour
  async ({ event, step, logger }) => {
    const db = connectDatabase();
    const resend = new Resend({ apiKey: env().RESEND_API_KEY });

    const workspaces = await step.run("list workspaces", () =>
      db.query.workspaces.findMany({
        where: (table, { isNotNull, lte, and }) =>
          and(isNotNull(table.trialEnds), lte(table.trialEnds, new Date())),
      }),
    );

    for (const ws of workspaces) {
      await step.run("end trial", async () =>
        db
          .update(schema.workspaces)
          .set({
            trialEnds: null,
            plan: "free",
            subscriptions: null,
            maxActiveKeys: 100, // TODO: read from constant
            maxVerifications: 2500,
          })
          .where(eq(schema.workspaces.id, ws.id)),
      );

      const users = await step.run("get users for workspace", () => getUsers(ws.tenantId));

      for await (const user of users) {
        logger.info(`sending trial ended email to ${user.email}`);
        await resend.sendTrialEnded({
          email: user.email,
          name: user.name,
          workspace: ws.name,
        });
      }
    }

    return {
      event,
      body: "done",
    };
  },
);

async function getUsers(tenantId: string): Promise<{ id: string; email: string; name: string }[]> {
  const userIds: string[] = [];
  if (tenantId.startsWith("org_")) {
    const members = await clerkClient.organizations.getOrganizationMembershipList({
      organizationId: tenantId,
    });
    for (const m of members) {
      userIds.push(m.publicUserData!.userId);
    }
  } else {
    userIds.push(tenantId);
  }

  return await Promise.all(
    userIds.map(async (userId) => {
      const user = await clerkClient.users.getUser(userId);
      const email = user.emailAddresses.at(0)?.emailAddress;
      if (!email) {
        throw new Error(`user ${user.id} does not have an email`);
      }
      return {
        id: user.id,
        name: user.firstName ?? user.username ?? "there",
        email,
      };
    }),
  );
}
