import { getClerkOrganizationsAdmins } from "@/lib/clerk";
import { type Budget, connectDatabase, inArray, schema } from "@/lib/db";
import { env } from "@/lib/env";
import { Tinybird } from "@/lib/tinybird";
import { chunkArray } from "@/lib/utils";
import { client } from "@/trigger";
import { cronTrigger } from "@trigger.dev/sdk";
import { getCurrentUsageBill } from "@unkey/billing";
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
        ws.budgets.filter(isActiveBudget).length > 0,
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
    // TODO: Find a way to pipe Tinybird for a given set of workspaceIds.
    const ratelimits = await io.runTask("list involved workspaces ratelimits", async () =>
      Promise.all(
        workspaces.map(async (workspace) =>
          tinybird
            .ratelimits({
              workspaceId: workspace.id,
              year,
              month,
            })
            .then((res) => ({
              ratelimits: res.data.at(0)?.success ?? 0,
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
      const usedRatelimits =
        ratelimits.find((usage) => usage.workspaceId === workspace.id)?.ratelimits || 0;

      const { currentPrice, error } = getCurrentUsageBill(
        {
          activeKeys: usedActiveKeys,
          verifications: usedVerifications,
          ratelimits: usedRatelimits,
        },
        workspace.subscriptions,
      );

      if (error) {
        io.logger.error(error, { workspaceId: workspace.id });
        throw new Error("getCurrentUsageBill failed");
      }

      return {
        currentPrice,
        workspaceId: workspace.id,
      };
    });

    io.logger.info("list workspaces that exceeded their budgeted usage");
    const exceededBudgetedAmount: {
      budgetId: string;
      emails: string[];
      tenantId: string;
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

      const budget = workspace.budgets
        .filter(isActiveBudget)
        .reduce<(typeof workspace.budgets)[number] | null>((acc, budget) => {
          // Check if current price exceeds the greatest budgeted amount
          if (
            currentPrice >= budget.fixedAmount &&
            (!acc || budget.fixedAmount > acc.fixedAmount)
          ) {
            return budget;
          }

          return acc;
        }, null);

      if (budget) {
        exceededBudgetedAmount.push({
          budgetId: budget.id,
          emails: budget.data.additionalEmails ?? [],
          tenantId: workspace.tenantId,
          workspaceId: workspace.id,
          workspaceName: workspace.name,
          budgetedAmount: budget.fixedAmount,
          currentPeriodBilling: currentPrice,
        });
      }
    });

    // TODO: Assumming admin users will not change frequently their emails, we could store in in the budget entity to avoid this step.
    await io.runTask(
      "retrieve workpaces admin emails from Clerk for exceeded budgets",
      async () => {
        const uniqueTenantsIds = exceededBudgetedAmount.reduce<string[]>((acc, { tenantId }) => {
          if (!acc.includes(tenantId)) {
            acc.push(tenantId);
          }

          return acc;
        }, []);

        const admins = await getClerkOrganizationsAdmins(uniqueTenantsIds);

        for (let i = 0; i < exceededBudgetedAmount.length; i++) {
          const foundAdmin = admins.find(
            (admin) => admin.tenantId === exceededBudgetedAmount[i].tenantId,
          );
          if (foundAdmin) {
            exceededBudgetedAmount[i].emails.unshift(foundAdmin.email);
          }
        }
      },
    );

    io.logger.info(`sending budget exceeded email to ${exceededBudgetedAmount.length} users`);

    const notifiedBudgetIds = await io.runTask(
      "sending budget exceeded email to involved workspaces",
      async () => {
        const _notifiedBudgetIds = [];

        // Whenever we failed to find the Admin email and no additional emails were set.
        const filteredExceededBudgetedAmount = exceededBudgetedAmount.filter(
          (budget) => budget.emails.length > 0,
        );
        const batches = chunkArray(filteredExceededBudgetedAmount, 100);

        for await (const batch of batches) {
          io.logger.info(`batch sending budget exceeded email to ${batch.length} users`);

          const { success } = await resend.sendBatchBudgetExceeded(
            batch.map((budget) => ({
              ...budget,
              email: budget.emails[0], /// @dev `budget.emails` always have at least 1 element.
              ccEmails: budget.emails.slice(1),
            })),
          );

          if (success) {
            _notifiedBudgetIds.push(...batch.map((budget) => budget.budgetId));
          } else {
            io.logger.error("failed to batch send budget exceeded email", {
              budgetIds: batch.map((budget) => budget.budgetId),
            });
          }
        }

        return _notifiedBudgetIds;
      },
    );

    await io.runTask(
      `updating ${notifiedBudgetIds.length} notified budgets in database`,
      async () => {
        await db
          .update(schema.budgets)
          .set({ lastReachedAt: new Date() })
          .where(inArray(schema.budgets.id, notifiedBudgetIds));
      },
    );

    return {};
  },
});

/**
 * Determines if a budget is currently enabled and has not reached its limit this month.
 * This implies the budget has not been notified for the current month.
 */
function isActiveBudget(budget: Budget) {
  const currentDate = new Date();
  const currentMonth = currentDate.getUTCMonth();
  const currentYear = currentDate.getUTCFullYear();

  const lastReachedDate = budget.lastReachedAt ? new Date(budget.lastReachedAt) : null;
  const isSameMonthAndYear =
    lastReachedDate &&
    lastReachedDate.getUTCMonth() === currentMonth &&
    lastReachedDate.getUTCFullYear() === currentYear;

  return budget.enabled && !isSameMonthAndYear;
}
