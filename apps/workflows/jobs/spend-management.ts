import { connectDatabase } from "@/lib/db";
import { env } from "@/lib/env";
import { Tinybird } from "@/lib/tinybird";
import { client } from "@/trigger";
import { cronTrigger } from "@trigger.dev/sdk";
import { calculateTieredPrices } from "@unkey/billing";
import { Resend } from "@unkey/resend";

client.defineJob({
  id: "billing.budgets.usage-reached",
  name: "Spend Management",
  version: "0.0.1",
  trigger: cronTrigger({
    cron: "*/10 * * * *",
  }),
  run: async (_payload, io, _ctx) => {
    const db = connectDatabase();
    const tinybird = new Tinybird(env().TINYBIRD_TOKEN);
    const resend = new Resend({ apiKey: env().RESEND_API_KEY });

    let workspaces = await io.runTask("list workspaces", async () =>
      db.query.workspaces.findMany({
        where: (table, { isNotNull, isNull, not, eq, and }) =>
          and(
            isNotNull(table.stripeCustomerId),
            isNotNull(table.subscriptions),
            not(eq(table.plan, "free")),
            isNull(table.deletedAt),
          ),
        with: {
          budgets: true,
        },
      }),
    );

    workspaces = workspaces.filter(
      (ws) =>
        ws.subscriptions &&
        Object.keys(ws.subscriptions).length > 0 &&
        ws.budgets.filter((budget) => budget.enabled).length > 0,
    );

    io.logger.info(`found ${workspaces.length} workspaces`, workspaces);

    const t = new Date();
    t.setUTCMonth(t.getUTCMonth() - 1);
    const year = t.getUTCFullYear();
    const month = t.getUTCMonth() + 1; // months are 0 indexed

    // TODO: Find a way to pipe Tinybird for a given set of workspaceIds.
    const activeKeys = await io.runTask("list involved workspaces active keys", async () =>
      Promise.all(
        workspaces.map(async (workspace) =>
          tinybird
            .activeKeys({
              workspaceId: workspace.id,
              year,
              month,
            })
            .then((res) => ({
              activeKeys: res.data.at(0)?.keys ?? 0,
              workspaceId: workspace.id,
            })),
        ),
      ),
    );

    // TODO: Find a way to pipe Tinybird for a given set of workspaceIds.
    const verifications = await io.runTask("list involved workspaces key verifications", async () =>
      Promise.all(
        workspaces.map(async (workspace) =>
          tinybird
            .verifications({
              workspaceId: workspace.id,
              year,
              month,
            })
            .then((res) => ({
              verifications: res.data.at(0)?.success ?? 0,
              workspaceId: workspace.id,
            })),
        ),
      ),
    );

    io.logger.info("estimate workspaces current billing");
    const currentUsages = workspaces.map((workspace) => {
      const usedActiveKeys =
        activeKeys.find((usage) => usage.workspaceId === workspace.id)?.activeKeys || 0;
      const usedVerifications =
        verifications.find((usage) => usage.workspaceId === workspace.id)?.verifications || 0;

      let currentPrice = 0;
      if (workspace.subscriptions?.plan) {
        const cost = parseFloat(workspace.subscriptions.plan.cents);
        currentPrice += cost;
      }
      if (workspace.subscriptions?.support) {
        const cost = parseFloat(workspace.subscriptions.support.cents);
        currentPrice += cost;
      }
      if (workspace.subscriptions?.activeKeys) {
        const cost = calculateTieredPrices(
          workspace.subscriptions.activeKeys.tiers,
          usedActiveKeys,
        );
        // TODO: What to do on error? Skip billing? Assume zero?
        if (!cost.error) {
          currentPrice += cost.value.totalCentsEstimate;
        }
      }
      if (workspace.subscriptions?.verifications) {
        const cost = calculateTieredPrices(
          workspace.subscriptions.verifications.tiers,
          usedVerifications,
        );
        // TODO: What to do on error? Skip billing? Assume zero?
        if (!cost.error) {
          currentPrice += cost.value.totalCentsEstimate;
        }
      }

      return {
        currentPrice,
        workspaceId: workspace.id,
      };
    });

    io.logger.info("list workspaces that exceeded their budgeted usage");
    const exceededBudgetedAmount: {
      email: string;
      workspaceId: string;
      workspaceName: string;
      budgetedAmount: number;
      currentPeriodBilling: number;
    }[] = [];
    workspaces.forEach((workspace) => {
      const currentPrice = currentUsages.find(
        (usage) => usage.workspaceId === workspace.id,
      )?.currentPrice;
      if (!currentPrice) {
        return null;
      }

      const budget = workspace.budgets.reduce<(typeof workspace.budgets)[number] | null>(
        (acc, budget) => {
          // Check if current price exceeds the greatest budgeted amount
          if (
            currentPrice >= budget.fixedAmount &&
            (!acc || budget.fixedAmount > acc.fixedAmount)
          ) {
            return budget;
          }

          return acc;
        },
        null,
      );

      if (budget) {
        // TODO: Extract the email of the workspace admin.
        // ->

        budget.data.additionalEmails?.forEach((email) => {
          exceededBudgetedAmount.push({
            email,
            workspaceId: workspace.id,
            workspaceName: workspace.name,
            budgetedAmount: budget.fixedAmount,
            currentPeriodBilling: currentPrice,
          });
        });
      }
    });

    io.logger.info(`sending budget exceeded email to ${exceededBudgetedAmount.length} users`);
    const batches = chunkArray(exceededBudgetedAmount, 100);
    for await (const batch of batches) {
      io.logger.info(`batch sending budget exceeded email to ${batch.length} users`);

      await resend.sendBatchBudgetExceeded(batch);
    }

    return {};
  },
});

// TODO: This could be a centralized util function.
function chunkArray<T>(array: T[], chunkSize: number): T[][] {
  const result: T[][] = [];
  for (let i = 0; i < array.length; i += chunkSize) {
    const chunk = array.slice(i, i + chunkSize);
    result.push(chunk);
  }
  return result;
}
