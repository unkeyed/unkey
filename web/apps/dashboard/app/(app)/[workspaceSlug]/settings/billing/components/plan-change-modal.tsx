"use client";

import { formatDollars } from "@/lib/fmt";
import { Button, DialogContainer } from "@unkey/ui";
import { cn } from "cn";
import { useEffect, useId, useState } from "react";

export type PlanOption = {
  id: string;
  name: string;
  /** Fee in the smallest currency unit (cents), or null for "Contact us". */
  amount: number | null;
  /** Recurring interval ("month"), or null when unknown. */
  interval: string | null;
  /** Optional muted detail after the name, e.g. "250K requests/month". */
  detail?: string;
};

/** "/mo" reads better than "/month" next to a price. */
function intervalSuffix(interval: string | null): string {
  switch (interval) {
    case "month":
      return "/mo";
    case "year":
      return "/yr";
    default:
      return interval ? `/${interval}` : "";
  }
}

type PlanChangeModalProps = {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  subTitle: string;
  options: PlanOption[];
  currentId: string | null;
  /**
   * Optional warning shown under the list for the highlighted option, e.g.
   * "your usage already exceeds this plan's credits". Return null for none.
   */
  warningFor?: (selected: PlanOption) => string | null;
  /** Footnote under the CTA when changing plans (proration semantics). */
  changeNote?: string;
  /** The option whose mutation is in flight, for the CTA loading state. */
  submittingId: string | undefined;
  onSelect: (id: string) => void;
  cancelAction?: React.ReactNode;
};

/**
 * One plan picker for every product on the billing page, so Compute and API
 * plan changes look and behave identically: radio rows with the plan detail
 * inline and the fee right-aligned, a directional CTA (upgrade/downgrade),
 * and an optional warning when the highlighted plan does not cover the
 * period's usage.
 */
export const PlanChangeModal: React.FC<PlanChangeModalProps> = ({
  isOpen,
  onOpenChange,
  title,
  subTitle,
  options,
  currentId,
  warningFor,
  changeNote,
  submittingId,
  onSelect,
  cancelAction,
}) => {
  const [selected, setSelected] = useState<string | null>(currentId);

  // Reset the highlighted plan to the current one whenever the modal opens.
  useEffect(() => {
    if (isOpen) {
      setSelected(currentId);
    }
  }, [isOpen, currentId]);

  const isSubmitting = submittingId !== undefined;
  const ctaDisabled = !selected || selected === currentId || isSubmitting;

  const currentOption = options.find((o) => o.id === currentId);
  const selectedOption = options.find((o) => o.id === selected);

  // Directional CTA: changing plans is an upgrade or a downgrade, and the
  // button should say which. Falls back to "Change plan" when either fee is
  // unknown (e.g. "Contact us" plans).
  let ctaLabel = "Subscribe";
  if (currentId) {
    ctaLabel = "Change plan";
    if (
      selectedOption &&
      selectedOption.id !== currentId &&
      selectedOption.amount !== null &&
      currentOption?.amount != null
    ) {
      ctaLabel =
        selectedOption.amount > currentOption.amount
          ? `Upgrade to ${selectedOption.name}`
          : `Downgrade to ${selectedOption.name}`;
    }
  }

  const warning =
    selectedOption && selectedOption.id !== currentId && warningFor
      ? warningFor(selectedOption)
      : null;

  return (
    <DialogContainer
      isOpen={isOpen}
      onOpenChange={onOpenChange}
      title={title}
      subTitle={subTitle}
      footer={
        <div className="flex w-full flex-col items-center gap-2">
          <Button
            type="button"
            variant="primary"
            size="xlg"
            className="w-full rounded-lg"
            loading={isSubmitting}
            disabled={ctaDisabled}
            onClick={() => {
              if (selected) {
                onSelect(selected);
              }
            }}
          >
            {ctaLabel}
          </Button>
          {currentId && changeNote ? <div className="text-gray-9 text-xs">{changeNote}</div> : null}
          {cancelAction}
        </div>
      }
    >
      <div className="flex flex-col gap-3">
        <PlanOptionList
          options={options}
          currentId={currentId}
          selectedId={selected}
          onSelect={setSelected}
        />

        {warning ? <p className="text-sm text-warning-11 leading-5">{warning}</p> : null}
      </div>
    </DialogContainer>
  );
};

type PlanOptionListProps = {
  options: PlanOption[];
  currentId: string | null;
  selectedId: string | null;
  onSelect: (id: string) => void;
};

export function PlanOptionList({ options, currentId, selectedId, onSelect }: PlanOptionListProps) {
  const name = useId();
  return (
    <fieldset className="flex flex-col divide-y overflow-hidden rounded-xl border bg-raised">
      {options.map((option) => {
        const isCurrent = option.id === currentId;
        const isSelected = option.id === selectedId;
        return (
          <label
            key={option.id}
            className={cn(
              "flex cursor-pointer items-center gap-3 px-4 py-3 transition-colors duration-150 ease-out",
              "has-[:focus-visible]:outline has-[:focus-visible]:outline-2 has-[:focus-visible]:-outline-offset-2 has-[:focus-visible]:outline-accent-8",
              isSelected ? "bg-grayA-2" : "hover:bg-grayA-2",
            )}
          >
            <input
              type="radio"
              name={name}
              value={option.id}
              checked={isSelected}
              onChange={() => onSelect(option.id)}
              className="sr-only"
            />
            <span
              aria-hidden="true"
              className={cn(
                "flex size-4 shrink-0 items-center justify-center rounded-full border transition-colors duration-150 ease-out",
                isSelected ? "border-gray-12 bg-gray-12" : "border-grayA-6",
              )}
            >
              <span
                className={cn(
                  "size-1.5 rounded-full bg-gray-1 transition-transform duration-150 ease-out",
                  isSelected ? "scale-100" : "scale-0",
                )}
              />
            </span>
            <span className="flex min-w-0 flex-1 items-center gap-2">
              <span className="shrink-0 font-medium text-gray-12 text-sm tabular-nums">
                {option.name}
              </span>
              {option.detail ? (
                <span className="truncate text-gray-11 text-xs">{option.detail}</span>
              ) : null}
              {isCurrent ? (
                <span className="shrink-0 rounded-full bg-grayA-3 px-2 text-2xs text-gray-11 leading-4">
                  Current
                </span>
              ) : null}
            </span>
            <span className="shrink-0 text-gray-12 text-sm tabular-nums">
              {option.amount !== null ? (
                <>
                  {formatDollars(option.amount)}
                  <span className="text-gray-11 text-xs">{intervalSuffix(option.interval)}</span>
                </>
              ) : (
                "Contact us"
              )}
            </span>
          </label>
        );
      })}
    </fieldset>
  );
}
