"use client";

import { Switch } from "@/components/ui/switch";
import { formatDollars } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import { Button, DialogContainer, FormInput, toast } from "@unkey/ui";
import { useState } from "react";

/** Alert thresholds as fractions of the budget. */
export const ALERT_STEPS = [0.5, 0.75] as const;

/** Mirrors MAX_BUDGET_CENTS in the deploy-budget router so an over-cap value
 *  fails client-side with a readable message. */
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

/** The spend bar's fill and severity: neutral, amber from 75%, red at 100%.
 *  Suspended means the cap was reached, so the bar reads full-and-red even
 *  while the usage query lags behind it. */
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

export function SpendBudgetDialog({ open, onOpenChange }: SpendBudgetDialogProps) {
  const trpcUtils = trpc.useUtils();

  const { data: budget } = trpc.billing.getDeployBudget.useQuery(undefined, {
    staleTime: 30_000,
  });

  const currentBudget = budget?.budgetCents ?? null;
  const hasBudget = currentBudget !== null;

  // null = untouched, fall through to the saved budget; only edits are stored,
  // so a background refetch can't wipe input mid-edit.
  const [budgetDraft, setBudgetDraft] = useState<string | null>(null);
  const [stopDraft, setStopDraft] = useState<boolean | null>(null);
  const budgetInput = budgetDraft ?? (currentBudget != null ? String(currentBudget / 100) : "");
  const stopAtBudget = stopDraft ?? budget?.stopAtBudget ?? false;

  const setOpen = (value: boolean) => {
    if (!value) {
      setBudgetDraft(null);
      setStopDraft(null);
    }
    onOpenChange(value);
  };

  const save = trpc.billing.setDeployBudget.useMutation({
    onSuccess: async () => {
      setOpen(false);
      toast.success("Spend budget saved");
      // workspace.getCurrent carries deploySpendSuspended, which drives the
      // app-wide paused banner.
      await Promise.all([
        trpcUtils.billing.getDeployBudget.invalidate(),
        trpcUtils.workspace.getCurrent.invalidate(),
      ]);
    },
    onError: (err) => toast.error(err.message),
  });

  const budgetCents = parseDollars(budgetInput);
  const invalid = budgetCents === undefined || (stopAtBudget && budgetCents === null);

  return (
    <DialogContainer
      isOpen={open}
      onOpenChange={setOpen}
      title="Compute spend budget"
      subTitle="Applies to usage spend per calendar month."
      footer={
        <div className="flex w-full items-center justify-between gap-4">
          {hasBudget ? (
            <Button
              type="button"
              variant="ghost"
              color="danger"
              size="md"
              disabled={save.isLoading}
              onClick={() => save.mutate({ budgetCents: null, stopAtBudget: false })}
            >
              Remove budget
            </Button>
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
            setBudgetDraft(next);
            // Clearing the budget clears the stop too: a stop without a
            // budget has no trigger point.
            if (parseDollars(next) === null) {
              setStopDraft(false);
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
            onCheckedChange={setStopDraft}
            disabled={budgetCents === null || budgetCents === undefined}
          />
        </div>
      </div>
    </DialogContainer>
  );
}
