"use client";

import { formatDollars } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { Button, InfoTooltip } from "@unkey/ui";
import { ComputePausedBadge, PausedDocsLink, pausedBody } from "./compute-paused";
import { ADMIN_ONLY_TOOLTIP } from "./constants";
import { ALERT_STEPS, SpendBudgetDialog, spendBar } from "./spend-budget";

type SpendManagementProps = {
  /** Month-to-date gross usage spend in cents, or null while loading. */
  usageCents: number | null;
  isAdmin: boolean;
  /** Edit dialog open state, owned by the parent card so the header trigger
   *  and this section's Configure button share one dialog. */
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

/**
 * Alternative spend layout, inspired by Vercel's Billing settings: a titled
 * "Spend management" section with a Configure action in the header and a
 * bordered summary of the limit (amount, severity bar). Reuses the shared
 * budget dialog and bar-severity logic from spend-budget.tsx. When paused, the
 * summary carries the whole story: the header Paused badge next to the limit
 * label, a full-and-red bar, and the paused copy underneath.
 */
export function SpendManagement({ usageCents, isAdmin, open, onOpenChange }: SpendManagementProps) {
  const { data: budget } = trpc.billing.getDeployBudget.useQuery(undefined, {
    staleTime: 30_000,
  });

  const currentBudget = budget?.budgetCents ?? null;
  const hasBudget = currentBudget !== null;

  const suspended = budget?.suspended ?? false;
  const budgetLabel = hasBudget ? formatDollars(currentBudget) : undefined;

  const { fraction, fillClassName } = spendBar(usageCents, currentBudget, suspended);
  const percent =
    usageCents !== null && currentBudget ? Math.floor((usageCents / currentBudget) * 100) : null;

  const configure = (
    <InfoTooltip content={ADMIN_ONLY_TOOLTIP} disabled={isAdmin} asChild>
      <span>
        <Button variant="outline" size="md" disabled={!isAdmin} onClick={() => onOpenChange(true)}>
          {hasBudget ? "Configure" : "Add spend limit"}
        </Button>
      </span>
    </InfoTooltip>
  );

  return (
    <>
      <div className="-mx-5 flex flex-col gap-4 border-grayA-3 border-t px-5 pt-4">
        <div className="flex items-start justify-between gap-4">
          <div className="flex flex-col gap-0.5">
            <span className="font-medium text-[13px] text-gray-12">Spend management</span>
            <span className="text-[12px] text-gray-10">
              Manage what happens when your usage spend reaches a monthly limit.
            </span>
          </div>
          {configure}
        </div>

        {hasBudget ? (
          <>
            <div className="flex min-w-0 flex-col gap-1">
              <div className="flex items-center gap-2">
                <span className="text-[13px] text-gray-11">Spend limit</span>
                {suspended ? <ComputePausedBadge /> : null}
              </div>
              <span className="font-medium text-[13px] text-gray-12 tabular-nums">
                {usageCents !== null ? formatDollars(usageCents) : "—"} of{" "}
                {formatDollars(currentBudget)}
                {percent !== null ? ` (${percent}%)` : ""}
              </span>
            </div>
            <div className="relative h-1.5 w-full overflow-hidden rounded-full bg-grayA-3">
              {fraction !== null ? (
                <div
                  className={cn(
                    "h-full rounded-full transition-[width] duration-300",
                    fillClassName,
                  )}
                  style={{ width: `${fraction * 100}%` }}
                />
              ) : null}
              {ALERT_STEPS.map((step) => (
                <div
                  key={step}
                  className="absolute top-0 h-full w-px bg-gray-8"
                  style={{ left: `${step * 100}%` }}
                />
              ))}
            </div>
            {suspended ? (
              <p className="text-[13px] text-gray-11 leading-5">
                {pausedBody(budgetLabel)} <PausedDocsLink />
              </p>
            ) : null}
          </>
        ) : (
          <div className="rounded-lg border border-grayA-4 px-4 py-3">
            <p className="text-[13px] text-gray-10">
              <span className="text-gray-11">No spend limit set.</span> Cap monthly usage spend to
              get alerts and optionally stop workloads.
            </p>
          </div>
        )}
      </div>

      <SpendBudgetDialog open={open} onOpenChange={onOpenChange} />
    </>
  );
}
