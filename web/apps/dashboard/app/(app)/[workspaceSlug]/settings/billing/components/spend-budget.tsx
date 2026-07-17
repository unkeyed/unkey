"use client";

import { Switch } from "@/components/ui/switch";
import { formatDollars } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import { Button, DialogContainer, FormInput, toast } from "@unkey/ui";
import { useEffect, useState } from "react";

/** Alert thresholds as fractions of the budget; fixed, like Vercel's. */
export const ALERT_STEPS = [0.5, 0.75] as const;

/**
 * Mirrors MAX_BUDGET_CENTS in the deploy-budget router so an over-cap value
 * fails client-side with a readable message instead of surfacing the server's
 * raw validation error.
 */
const MAX_BUDGET_CENTS = 1_000_000_000;

/**
 * Parses a whole-dollar form value into cents. Empty = no budget (null);
 * anything non-numeric, non-positive, or above the server's budget cap is
 * invalid (undefined).
 */
function parseDollars(value: string): number | null | undefined {
  const trimmed = value.trim();
  if (trimmed === "") {
    return null;
  }
  if (!/^\d+$/.test(trimmed)) {
    return undefined;
  }
  const cents = Number.parseInt(trimmed, 10) * 100;
  return cents > 0 && cents <= MAX_BUDGET_CENTS ? cents : undefined;
}

/**
 * The spend bar's fill fraction and severity color. Paused means the spend cap
 * was reached, so the meter reads full-and-red even when a forced preview has
 * usage sitting below the budget. Severity steps: neutral, amber from 75%, red
 * at 100%.
 */
export function spendBar(
  usageCents: number | null,
  budgetCents: number | null,
  suspended: boolean,
): { fraction: number | null; fillClassName: string } {
  const fraction = suspended
    ? 1
    : usageCents !== null && budgetCents
      ? Math.min(1, Math.max(0, usageCents / budgetCents))
      : null;
  const usedFraction = usageCents !== null && budgetCents ? usageCents / budgetCents : 0;
  const fillClassName =
    suspended || usedFraction >= 1
      ? "bg-error-9"
      : usedFraction >= 0.75
        ? "bg-warning-9"
        : "bg-gray-9";
  return { fraction, fillClassName };
}

type SpendBudgetDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

/**
 * The budget edit dialog: a monthly budget amount plus a "stop workloads"
 * switch. Self-contained (its own copy of the budget query and form state);
 * React Query dedupes the query to one request.
 */
export function SpendBudgetDialog({ open, onOpenChange }: SpendBudgetDialogProps) {
  const trpcUtils = trpc.useUtils();
  const [budgetInput, setBudgetInput] = useState("");
  const [stopAtBudget, setStopAtBudget] = useState(false);

  const { data: budget } = trpc.billing.getDeployBudget.useQuery(undefined, {
    staleTime: 30_000,
  });

  const save = trpc.billing.setDeployBudget.useMutation({
    onSuccess: async () => {
      onOpenChange(false);
      toast.success("Spend budget saved");
      await trpcUtils.billing.getDeployBudget.invalidate();
    },
    onError: (err) => toast.error(err.message),
  });

  const budgetCents = parseDollars(budgetInput);
  const invalid = budgetCents === undefined || (stopAtBudget && budgetCents === null);

  const currentBudget = budget?.budgetCents ?? null;
  const hasBudget = currentBudget !== null;

  // biome-ignore lint/correctness/useExhaustiveDependencies: seed the form only when the dialog opens, not when a background refetch lands mid-edit
  useEffect(() => {
    if (open) {
      setBudgetInput(currentBudget != null ? String(currentBudget / 100) : "");
      setStopAtBudget(budget?.stopAtBudget ?? false);
    }
  }, [open]);

  return (
    <DialogContainer
      isOpen={open}
      onOpenChange={onOpenChange}
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
          description="We email you when your usage spend reaches 50%, 75% and 100% of this amount. Leave empty for no budget."
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
            budgetCents === undefined
              ? "Enter a whole dollar amount up to $10,000,000, or leave empty."
              : undefined
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
  );
}
