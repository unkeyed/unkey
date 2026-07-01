"use client";

import { formatDollars } from "@/lib/fmt";
import { cn } from "@/lib/utils";
import { Button, DialogContainer } from "@unkey/ui";
import { useEffect, useState } from "react";

export type PlanOption = {
  id: string;
  name: string;
  /** Fee in the smallest currency unit (cents), or null for "Contact us". */
  amount: number | null;
  /** Recurring interval ("month"), or null when unknown. */
  interval: string | null;
  /** Short inline detail next to the name: "250K requests/month". */
  detail: string;
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
        </div>
      }
    >
      <div className="flex flex-col gap-3">
        {options.map((option) => {
          const isCurrent = option.id === currentId;
          const isSelected = option.id === selected;
          return (
            <button
              type="button"
              key={option.id}
              onClick={() => setSelected(option.id)}
              className={cn(
                "w-full rounded-lg border px-4 py-2 text-left transition-all",
                isSelected
                  ? "border-info-7 bg-info-2 ring-1 ring-info-7"
                  : isCurrent
                    ? "border-gray-5 bg-gray-2 hover:border-gray-6"
                    : "border-gray-4 hover:border-gray-6",
              )}
            >
              <div className="flex items-center justify-between gap-3 py-1">
                <div className="flex min-w-0 items-center gap-3">
                  <div
                    className={cn(
                      "flex size-4 shrink-0 items-center justify-center rounded-full border-2",
                      isSelected ? "border-info-9 bg-info-9" : "border-gray-6",
                    )}
                  >
                    {isSelected ? <div className="size-2 rounded-full bg-white" /> : null}
                  </div>
                  <span className="min-w-[120px] font-medium text-[15px] text-gray-12">
                    {option.name}
                  </span>
                  <span className="truncate text-[12px] text-gray-11">{option.detail}</span>
                  {isCurrent ? (
                    <span className="rounded-full bg-info-3 px-2 text-[11px] text-info-11 leading-4">
                      Current
                    </span>
                  ) : null}
                </div>
                <span className="shrink-0 font-medium text-[15px] text-gray-12 tabular-nums">
                  {option.amount !== null ? (
                    <>
                      {formatDollars(option.amount)}
                      <span className="font-normal text-[12px] text-gray-11">
                        {intervalSuffix(option.interval)}
                      </span>
                    </>
                  ) : (
                    "Contact us"
                  )}
                </span>
              </div>
            </button>
          );
        })}

        {warning ? <p className="text-[13px] text-warning-11 leading-5">{warning}</p> : null}
      </div>
    </DialogContainer>
  );
};
