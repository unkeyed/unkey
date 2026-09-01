"use client";

import { Switch } from "@/components/ui/switch";
import { formatDollars } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import {
  Button,
  DialogContainer,
  FormField,
  InputGroup,
  InputGroupInput,
  InputGroupText,
  toast,
} from "@unkey/ui";
import { useState } from "react";
import { ALERT_STEPS } from "./constants";

const ALERT_STEPS_LABEL = ALERT_STEPS.map((step) => `${step * 100}%`).join(", ");

const MAX_BUDGET_CENTS = 1_000_000_000;

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
        <FormField
          label="Monthly budget"
          description={`We email you when your usage spend reaches ${ALERT_STEPS_LABEL} of this amount. Leave empty for no budget.`}
          error={
            budgetCents === undefined
              ? "Enter a whole dollar amount up to $10,000,000, or leave empty."
              : undefined
          }
        >
          {(field) => (
            <InputGroup variant={field.variant}>
              <InputGroupText className="pl-2">$</InputGroupText>
              <InputGroupInput
                id={field.id}
                className="pl-px"
                placeholder="300"
                inputMode="numeric"
                value={budgetInput}
                aria-describedby={field.describedBy}
                aria-invalid={field.invalid}
                onChange={(e) => {
                  const next = e.currentTarget.value;
                  setBudgetDraft(next);
                  if (parseDollars(next) === null) {
                    setStopDraft(false);
                  }
                }}
              />
            </InputGroup>
          )}
        </FormField>
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
