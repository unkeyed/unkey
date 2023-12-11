import { connectDatabase, eq, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { client } from "@/trigger";
import { clerkClient } from "@clerk/nextjs";
import { cronTrigger } from "@trigger.dev/sdk";
import { Resend } from "@unkey/resend";

client.defineJob({
  id: "billing.trials.end",
  name: "End trials",
  version: "0.0.1",
  trigger: cronTrigger({
    cron: "0 * * * *",
  }),
  run: async (_payload, io, _ctx) => {
    const db = connectDatabase();
    const resend = new Resend({ apiKey: env().RESEND_API_KEY });

    const workspaces = await io.runTask("list workspaces", () =>
      db.query.workspaces.findMany({
        where: (table, { isNotNull, lte, and }) =>
          and(isNotNull(table.trialEnds), lte(table.trialEnds, new Date())),
      }),
    );
    io.logger.info(`found ${workspaces.length} workspaces with an expired trial`);

    for (const ws of workspaces) {
      await io.runTask(`end trial for worksapce ${ws.id}`, async () => {
        await db
          .update(schema.workspaces)
          .set({
            trialEnds: null,
            plan: "free",
            subscriptions: null,
            maxActiveKeys: 100, // TODO: read from constant
            maxVerifications: 2500,
          })
          .where(eq(schema.workspaces.id, ws.id));
      });

      const users = await io.runTask(`get users for workspace ${ws.id}`, () =>
        getUsers(ws.tenantId),
      );

      for await (const user of users) {
        io.logger.info(`sending trial ended email to ${user.email}`);
        await resend.sendTrialEnded({
          email: user.email,
          name: user.name,
          workspace: ws.name,
        });
      }
    }

    return {};
  },
});

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
