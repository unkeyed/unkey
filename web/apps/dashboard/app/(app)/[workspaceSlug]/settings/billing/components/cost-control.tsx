"use client";

import { formatDollars } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import { Ban, Cube, Envelope, Nodes } from "@unkey/icons";
import {
  Button,
  InfoTooltip,
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemFooter,
  ItemGroup,
  ItemHeader,
  ItemMedia,
  ItemSeparator,
  ItemTitle,
} from "@unkey/ui";
import { type ReactNode, useState } from "react";
import { ComputePausedBadge, PausedDocsLink, pausedBody } from "./compute-paused";
import { ADMIN_ONLY_TOOLTIP } from "./constants";
import { ALERT_STEPS, SpendBudgetDialog } from "./spend-budget";

type CostControlProps = {
  isAdmin: boolean;
  /** False when Deploy billing is not configured server-side, as on the plans card. */
  showCompute: boolean;
  /** The API plan's flat monthly fee in cents. */
  apiFeeCents: number;
};

/**
 * Split per product so the Compute budget cannot be read as workspace-wide: API
 * management sitting beside it with a fixed fee is what scopes it.
 */
export function CostControl({ isAdmin, showCompute, apiFeeCents }: CostControlProps) {
  return (
    <div className="flex flex-col gap-3">
      <h2 className="font-medium text-[13px] text-gray-12">Cost control</h2>
      {showCompute ? <ComputeBudget isAdmin={isAdmin} /> : null}
      <ApiFixedFee apiFeeCents={apiFeeCents} />
    </div>
  );
}

/**
 * The budget is the setting, so the budget is the figure. What has been spent
 * against it is a Usage question, not a Billing one.
 */
function ComputeBudget({ isAdmin }: { isAdmin: boolean }) {
  const [open, setOpen] = useState(false);

  const { data: budget } = trpc.billing.getDeployBudget.useQuery(undefined, { staleTime: 30_000 });

  const budgetCents = budget?.budgetCents ?? null;
  const suspended = budget?.suspended ?? false;
  const stopAtBudget = budget?.stopAtBudget ?? false;

  return (
    <>
      <ItemGroup variant="outline">
        <ItemHeader>
          <ItemMedia className="bg-orangeA-3 text-orange-11">
            <Cube />
          </ItemMedia>
          <ItemContent>
            <div className="flex h-4 items-center gap-2">
              <ItemTitle>Compute</ItemTitle>
              {suspended ? <ComputePausedBadge /> : null}
            </div>
          </ItemContent>
          <ItemActions>
            <InfoTooltip content={ADMIN_ONLY_TOOLTIP} disabled={isAdmin} asChild>
              <span>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={!isAdmin}
                  onClick={() => setOpen(true)}
                >
                  {budgetCents === null ? "Add" : "Configure"}
                </Button>
              </span>
            </InfoTooltip>
          </ItemActions>
        </ItemHeader>

        <div className="px-4 pb-4">
          <span className="font-semibold text-3xl text-gray-12 tabular-nums tracking-tight">
            {budgetCents === null ? "None" : formatDollars(budgetCents)}
          </span>
        </div>

        {budgetCents !== null ? (
          <>
            <ItemSeparator />
            <div className="bg-grayA-2 px-4 py-2 font-semibold text-[10px] text-gray-9 uppercase tracking-wider">
              Alerts
            </div>
            {ALERT_STEPS.map((step) => (
              <AlertRow key={step} icon={<Envelope />}>
                Email at {step * 100}% ({formatDollars(budgetCents * step)})
              </AlertRow>
            ))}
            {stopAtBudget ? (
              <AlertRow icon={<Ban />} mediaClassName="bg-errorA-3 text-error-11">
                Stop workloads at 100% ({formatDollars(budgetCents)})
              </AlertRow>
            ) : null}
          </>
        ) : null}

        {suspended ? (
          <>
            <ItemSeparator />
            <ItemFooter>
              <span>
                {pausedBody(budgetCents === null ? undefined : formatDollars(budgetCents))}{" "}
                <PausedDocsLink />
              </span>
            </ItemFooter>
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

/**
 * Carries no budget of its own, and saying so is what scopes the one above. It
 * states why rather than what happens past the quota: that policy is due to
 * change to a rate limit.
 */
function ApiFixedFee({ apiFeeCents }: { apiFeeCents: number }) {
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
          <ItemDescription>Fixed at {formatDollars(apiFeeCents)}/month.</ItemDescription>
        </ItemContent>
      </Item>
    </ItemGroup>
  );
}
