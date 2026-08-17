"use client";

import { formatDollars } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import { Ban, Cube, Envelope, Nodes } from "@unkey/icons";
import { P, match } from "@unkey/match";
import {
  AlertBanner,
  AlertBannerDescription,
  Button,
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemHeader,
  ItemMedia,
  ItemSeparator,
  ItemTitle,
  Skeleton,
} from "@unkey/ui";
import { type ReactNode, useState } from "react";
import { AdminGate } from "./admin-gate";
import { ALERT_STEPS } from "./constants";
import { SpendBudgetDialog } from "./spend-budget-v2";

type CostControlProps = {
  isAdmin: boolean | undefined;
  apiFeeCents: number | null;
};

export function CostControl({ isAdmin, apiFeeCents }: CostControlProps) {
  return (
    <div className="flex flex-col gap-3">
      <h2 className="font-medium text-[13px] text-gray-12">Cost control</h2>
      <ComputeBudget isAdmin={isAdmin} />
      <ApiFixedFee apiFeeCents={apiFeeCents} />
    </div>
  );
}

function ComputeBudget({ isAdmin }: { isAdmin: boolean | undefined }) {
  const [open, setOpen] = useState(false);

  const {
    data: budget,
    isLoading,
    isError,
  } = trpc.billing.getDeployBudget.useQuery(undefined, { staleTime: 30_000 });

  const budgetCents = budget?.budgetCents ?? null;
  const stopAtBudget = budget?.stopAtBudget ?? false;
  const suspended = budget?.suspended ?? false;

  return (
    <>
      <ItemGroup variant="outline">
        {suspended ? (
          <div className="px-4 pt-4">
            <AlertBanner variant="warning">
              <AlertBannerDescription>
                Workloads are currently stopped.{" "}
                <AdminGate isAdmin={isAdmin}>
                  {(disabled) => (
                    <button
                      type="button"
                      disabled={disabled}
                      onClick={() => setOpen(true)}
                      className="cursor-pointer underline underline-offset-2 hover:opacity-80 disabled:cursor-not-allowed disabled:no-underline"
                    >
                      Configure
                    </button>
                  )}
                </AdminGate>
              </AlertBannerDescription>
            </AlertBanner>
          </div>
        ) : null}
        <ItemHeader>
          <ItemMedia className="bg-orangeA-3 text-orange-11">
            <Cube />
          </ItemMedia>
          <ItemContent>
            <ItemTitle>Compute</ItemTitle>
          </ItemContent>
          <ItemActions>
            <AdminGate isAdmin={isAdmin}>
              {(disabled) => (
                <Button
                  variant="outline"
                  size="sm"
                  disabled={disabled || isLoading || isError}
                  onClick={() => setOpen(true)}
                >
                  {isLoading || isError || budgetCents !== null ? "Configure" : "Add"}
                </Button>
              )}
            </AdminGate>
          </ItemActions>
        </ItemHeader>

        <div className="px-4 pb-4">
          {match({ isLoading, isError, budgetCents })
            .with({ isLoading: true }, () => <Skeleton className="h-9 w-28" />)
            .with({ isError: true }, () => (
              <AlertBanner variant="warning">
                <AlertBannerDescription>
                  The spend controls could not be loaded.
                </AlertBannerDescription>
              </AlertBanner>
            ))
            .with({ budgetCents: P.number }, ({ budgetCents }) => (
              <span className="font-semibold text-3xl text-gray-12 tabular-nums tracking-tight">
                {formatDollars(budgetCents)}
              </span>
            ))
            .otherwise(() => (
              <AlertBanner>
                <AlertBannerDescription>No spend controls are enabled yet</AlertBannerDescription>
              </AlertBanner>
            ))}
        </div>

        {budgetCents !== null ? (
          <>
            <ItemSeparator />
            <div className="bg-grayA-2 px-4 py-2 font-semibold text-[10px] text-gray-9 uppercase tracking-wider">
              Alerts
            </div>
            {/* The deployspendcheck worker sends the stopped email instead of the 100% warning when stopping is on. */}
            {ALERT_STEPS.filter((step) => !(stopAtBudget && step === 1)).map((step) => (
              <AlertRow key={step} icon={<Envelope />}>
                Email at {step * 100}% ({formatDollars(budgetCents * step)})
              </AlertRow>
            ))}
            {stopAtBudget ? (
              <AlertRow icon={<Ban />} mediaClassName="bg-errorA-3 text-error-11">
                Stop workloads and email at 100% ({formatDollars(budgetCents)})
              </AlertRow>
            ) : null}
          </>
        ) : null}
      </ItemGroup>

      <SpendBudgetDialog open={open} onOpenChange={setOpen} />
    </>
  );
}

function AlertRow({
  icon,
  mediaClassName,
  children,
}: {
  icon: ReactNode;
  mediaClassName?: string;
  children: ReactNode;
}) {
  return (
    <>
      <ItemSeparator />
      <Item>
        <ItemMedia className={mediaClassName}>{icon}</ItemMedia>
        <ItemContent>
          <ItemTitle className="font-normal">{children}</ItemTitle>
        </ItemContent>
      </Item>
    </>
  );
}

function ApiFixedFee({ apiFeeCents }: { apiFeeCents: number | null }) {
  return (
    <ItemGroup variant="outline">
      <ItemHeader>
        <ItemMedia className="bg-infoA-3 text-info-11">
          <Nodes />
        </ItemMedia>
        <ItemContent>
          <ItemTitle>API management</ItemTitle>
        </ItemContent>
      </ItemHeader>
      <ItemSeparator />
      <Item>
        <ItemContent>
          <ItemDescription className="text-[13px] leading-5">
            {apiFeeCents === null
              ? "Plan fee unavailable."
              : `Fixed at ${formatDollars(apiFeeCents)}/month.`}
          </ItemDescription>
        </ItemContent>
      </Item>
    </ItemGroup>
  );
}
