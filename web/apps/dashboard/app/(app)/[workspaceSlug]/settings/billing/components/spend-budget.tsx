"use client";

import { Switch } from "@/components/ui/switch";
import { formatDollars } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { Button, DialogContainer, FormInput, InfoTooltip, toast } from "@unkey/ui";
import { useState } from "react";

const ADMIN_ONLY_TOOLTIP = "Admin access required to manage billing";

/** Alert thresholds as fractions of the budget; fixed, like Vercel's. */
const ALERT_STEPS = [0.5, 0.75] as const;

/**
 * Parses a whole-dollar form value into cents. Empty = no budget (null);
 * anything non-numeric or non-positive is invalid (undefined).
 */
function parseDollars(value: string): number | null | undefined {
  const trimmed = value.trim();
  if (trimmed === "") {
    return null;
  }
  if (!/^\d+$/.test(trimmed)) {
    return undefined;
  }
  const dollars = Number.parseInt(trimmed, 10);
  return dollars > 0 ? dollars * 100 : undefined;
}

type SpendBudgetProps = {
  isAdmin: boolean;
  /** Month-to-date Compute usage spend in cents, or null while loading. */
  usageCents: number | null;
};

/**
 * The Compute spend-budget row: a flush section under the usage meter showing
 * month-to-date usage spend against the monthly budget, severity-colored with
 * ticks at the fixed alert thresholds (Vercel's model: one number, alerts at
 * percentages of it, stopping workloads is a toggle). The bar spans the full
 * width like the usage meter above it, so the ticks line up; the Edit action
 * sits on the caption line below. Unset budget renders a one-line invitation
 * instead. When the spend cap has paused compute, a warning banner sits above
 * it.
 */
export const SpendBudget: React.FC<SpendBudgetProps> = ({ isAdmin, usageCents }) => {
  const trpcUtils = trpc.useUtils();
  const [isOpen, setOpen] = useState(false);
  const [budgetInput, setBudgetInput] = useState("");
  const [stopAtBudget, setStopAtBudget] = useState(false);

  const { data: budget } = trpc.billing.getDeployBudget.useQuery(undefined, {
    staleTime: 30_000,
  });

  const save = trpc.billing.setDeployBudget.useMutation({
    onSuccess: async () => {
      setOpen(false);
      toast.success("Spend budget saved");
      await trpcUtils.billing.getDeployBudget.invalidate();
    },
    onError: (err) => toast.error(err.message),
  });

  const budgetCents = parseDollars(budgetInput);
  const invalid = budgetCents === undefined || (stopAtBudget && budgetCents === null);

  const openDialog = () => {
    setBudgetInput(budget?.budgetCents != null ? String(budget.budgetCents / 100) : "");
    setStopAtBudget(budget?.stopAtBudget ?? false);
    setOpen(true);
  };

  const currentBudget = budget?.budgetCents ?? null;
  const hasBudget = currentBudget !== null;
  const suspended = budget?.suspended ?? false;

  const fraction =
    usageCents !== null && currentBudget
      ? Math.min(1, Math.max(0, usageCents / currentBudget))
      : null;
  const usedFraction = usageCents !== null && currentBudget ? usageCents / currentBudget : 0;
  // Severity steps like Vercel's ring: neutral, amber from 75%, red at 100%.
  const fillClassName =
    usedFraction >= 1 ? "bg-error-9" : usedFraction >= 0.75 ? "bg-warning-9" : "bg-gray-9";

  const editButton = (
    <InfoTooltip content={ADMIN_ONLY_TOOLTIP} disabled={isAdmin} asChild>
      <span>
        <Button variant="outline" size="sm" disabled={!isAdmin || !budget} onClick={openDialog}>
          {hasBudget ? "Edit" : "Set budget"}
        </Button>
      </span>
    </InfoTooltip>
  );

  return (
    <>
      {suspended ? (
        <div className="rounded-lg border border-warning-6 bg-warningA-2 px-4 py-3">
          <span className="text-[11px] text-warning-11 uppercase tracking-wide">
            Compute paused
          </span>
          <p className="mt-1 text-[13px] text-gray-12">
            Compute is paused: spend cap reached. Raise or remove your budget and Compute
            resumes automatically within about 15 minutes.
          </p>
        </div>
      ) : null}
      {hasBudget ? (
        <div className="flex w-full flex-col gap-2">
          <div className="flex items-baseline justify-between gap-4">
            <span className="text-[13px] text-gray-11">Spend budget</span>
            <span className="font-medium text-[13px] text-gray-12 tabular-nums">
              {usageCents !== null ? formatDollars(usageCents) : "—"} of{" "}
              {formatDollars(currentBudget)}
              {usageCents !== null && currentBudget
                ? ` (${Math.round((usageCents / currentBudget) * 100)}%)`
                : ""}
            </span>
          </div>
          <div className="relative h-1.5 w-full overflow-hidden rounded-full bg-grayA-3">
            {fraction !== null ? (
              <div
                className={cn("h-full rounded-full transition-[width] duration-300", fillClassName)}
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
          <div className="flex items-center justify-between gap-4">
            <span className="text-[12px] text-gray-10">
              {budget?.stopAtBudget
                ? "Email alerts at 50%, 75% and 100% · workloads stop at the budget"
                : "Email alerts at 50%, 75% and 100% · workloads keep running"}
            </span>
            <div className="shrink-0">{editButton}</div>
          </div>
        </div>
      ) : (
        <div className="flex items-center justify-between gap-4">
          <p className="text-[13px] text-gray-10">
            <span className="text-gray-11">No spend budget.</span> Get email alerts and optionally
            stop workloads at a monthly usage amount.
          </p>
          <div className="shrink-0">{editButton}</div>
        </div>
      )}

      <DialogContainer
        isOpen={isOpen}
        onOpenChange={setOpen}
        title="Compute spend budget"
        subTitle="Applies to usage spend per calendar month."
        footer={
          <div className="flex w-full items-center justify-between gap-4">
            {hasBudget ? (
              <button
                type="button"
                className="text-[13px] text-error-9 transition-colors hover:text-error-11"
                disabled={save.isLoading}
                onClick={() => save.mutate({ budgetCents: null, stopAtBudget: false })}
              >
                Remove budget
              </button>
            ) : (
              <span />
            )}
            <Button
              type="button"
              variant="primary"
              size="xlg"
              className="rounded-lg px-8"
              disabled={invalid}
              loading={save.isLoading}
              onClick={() => {
                if (budgetCents === undefined) {
                  return;
                }
                save.mutate({ budgetCents, stopAtBudget });
              }}
            >
              Save budget
            </Button>
          </div>
        }
      >
        <div className="flex flex-col gap-5">
          <FormInput
            label="Monthly budget"
            description="We email you when usage spend reaches 50%, 75% and 100% of this amount. Leave empty for no budget."
            placeholder="300"
            prefix="$"
            inputMode="numeric"
            value={budgetInput}
            onChange={(e) => {
              const next = e.currentTarget.value;
              setBudgetInput(next);
              // Clearing the budget clears the stop too: a stop without a
              // budget has no trigger point.
              if (parseDollars(next) === null) {
                setStopAtBudget(false);
              }
            }}
            error={
              budgetCents === undefined ? "Enter a whole dollar amount, or leave empty." : undefined
            }
          />
          <div className="flex items-start justify-between gap-4">
            <div className="flex flex-col gap-1">
              <span className="text-[13px] text-gray-12">Stop workloads at the budget</span>
              <span className="text-[12px] text-gray-10">
                {budgetCents != null
                  ? `Workloads stop for the rest of the month when usage spend reaches ${formatDollars(budgetCents)}.`
                  : "Workloads stop for the rest of the month when usage spend reaches the budget."}
              </span>
            </div>
            <Switch
              checked={stopAtBudget}
              onCheckedChange={setStopAtBudget}
              disabled={budgetCents === null || budgetCents === undefined}
            />
          </div>
        </div>
      </DialogContainer>
    </>
  );
};
