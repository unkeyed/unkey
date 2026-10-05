"use client";

import { FEATURE_ICONS } from "@/components/billing/plan-feature-icons";
import { PlanTierIcon } from "@/components/billing/plan-tier-icons";
import { currentApiProduct } from "@/lib/billing/api-plan";
import { planName } from "@/lib/billing/plan-card-state";
import { planFeatures } from "@/lib/billing/plan-features";
import { type UpgradeResult, parseUpgradeResult, upgradeQuery } from "@/lib/billing/upgrade-result";
import { formatNumber } from "@/lib/fmt";
import type { DeployPlan } from "@/lib/stripe/deployPlan";
import { trpc } from "@/lib/trpc/client";
import { useWorkspace } from "@/providers/workspace-provider";
import { IconNodesOutline18, type IconProps, IconUserOutline18 } from "@unkey/icons";
import { Button, Dialog, DialogContent, DialogDescription, DialogTitle } from "@unkey/ui";
import { usePathname, useSearchParams } from "next/navigation";
import { type ComponentType, type ReactNode, useEffect, useEffectEvent, useState } from "react";
import { SpinCoin } from "./spin-coin";

type Unlocked = { label: string; Icon: ComponentType<IconProps> };

type Celebration = {
  name: string;
  description: string;
  back: ReactNode;
  unlocked: Unlocked[];
};

const RESULT_PARAMS = ["upgraded", "plan"];

function computeCelebration(plan: DeployPlan): Celebration {
  return {
    name: planName(plan, undefined),
    description: "Your Compute plan is active. Here's what you unlocked.",
    back: <PlanTierIcon plan={plan} className="size-8" />,
    unlocked: planFeatures(plan)
      .filter((row) => row.included)
      .map((row) => ({ label: row.label, Icon: FEATURE_ICONS[row.kind] })),
  };
}

function apiCelebration(product: { name: string; requestsPerMonth: number }): Celebration {
  return {
    name: product.name,
    description: "Your API plan is active. Here's what you unlocked.",
    back: <IconNodesOutline18 className="size-8" />,
    unlocked: [
      {
        label: `${formatNumber(product.requestsPerMonth)} requests per month`,
        Icon: IconNodesOutline18,
      },
      { label: "Unlimited team members", Icon: IconUserOutline18 },
    ],
  };
}

export function announceUpgrade(result: UpgradeResult) {
  const url = new URL(window.location.href);
  for (const [name, value] of Object.entries(upgradeQuery(result))) {
    url.searchParams.set(name, value);
  }
  window.history.replaceState(null, "", url);
}

export function UpgradeSuccessDialog() {
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const [shown, setShown] = useState<{ id: number; result: UpgradeResult } | null>(null);

  useEffect(() => {
    const result = parseUpgradeResult(searchParams ?? new URLSearchParams());
    if (!result) {
      return;
    }
    const kept = new URLSearchParams(searchParams?.toString());
    for (const name of RESULT_PARAMS) {
      kept.delete(name);
    }
    const query = kept.toString();
    window.history.replaceState(null, "", `${pathname}${query ? `?${query}` : ""}`);
    setShown((previous) => ({ id: (previous?.id ?? 0) + 1, result }));
  }, [pathname, searchParams]);

  return shown ? <UpgradeCelebration key={shown.id} result={shown.result} /> : null;
}

function UpgradeCelebration({ result }: { result: UpgradeResult }) {
  switch (result.kind) {
    case "compute":
      return <ComputeUpgradeCelebration plan={result.plan} />;
    case "api":
      return <ApiUpgradeCelebration />;
  }
}

export function ComputeUpgradeCelebration({
  plan,
  onClose,
}: {
  plan: DeployPlan;
  onClose?: () => void;
}) {
  const subscriptionQuery = trpc.stripe.getDeploySubscription.useQuery(undefined, {
    staleTime: 0,
  });
  const matches = subscriptionQuery.data?.plan === plan;
  const settled = !subscriptionQuery.isFetching;
  const skip = useEffectEvent(() => onClose?.());
  useEffect(() => {
    if (settled && !matches) {
      skip();
    }
  }, [settled, matches]);
  if (!matches) {
    return null;
  }
  return <CelebrationDialog celebration={computeCelebration(plan)} onClose={onClose} />;
}

function ApiUpgradeCelebration() {
  const { data: billingInfo } = trpc.stripe.getBillingInfo.useQuery(undefined, {
    staleTime: 30_000,
    trpc: { context: { skipBatch: true } },
  });
  const product = billingInfo
    ? currentApiProduct({
        products: billingInfo.products,
        subscription: billingInfo.subscription,
        currentProductId: billingInfo.currentProductId,
      })
    : undefined;
  if (!product) {
    return null;
  }
  return (
    <CelebrationDialog
      celebration={apiCelebration({
        name: product.name,
        requestsPerMonth: product.quotas.requestsPerMonth,
      })}
    />
  );
}

function CelebrationDialog({
  celebration,
  onClose,
}: {
  celebration: Celebration;
  onClose?: () => void;
}) {
  const [open, setOpen] = useState(true);
  const firstName = useWorkspace().user?.firstName ?? null;
  return (
    <Dialog
      open={open}
      onOpenChange={setOpen}
      onOpenChangeComplete={(isOpen) => {
        if (!isOpen) {
          onClose?.();
        }
      }}
    >
      <DialogContent className="max-w-[440px] gap-0 overflow-hidden rounded-2xl! border-grayA-4 bg-raised p-0">
        <CoinHeader back={celebration.back} />
        <div className="relative flex flex-col gap-6 bg-raised p-6 pt-2">
          <div
            className="flex animate-upgrade-rise flex-col items-center gap-1 text-center motion-reduce:animate-none"
            style={{ animationDelay: "40ms" }}
          >
            <DialogTitle className="font-semibold text-gray-12 text-xl tracking-[-0.03em]">
              {firstName
                ? `Welcome to ${celebration.name}, ${firstName}!`
                : `Welcome to ${celebration.name}!`}
            </DialogTitle>
            <DialogDescription className="text-gray-11 text-sm leading-5">
              {celebration.description}
            </DialogDescription>
          </div>
          <ul className="flex flex-col gap-3">
            {celebration.unlocked.map(({ label, Icon }, i) => (
              <li
                key={label}
                className="flex animate-upgrade-rise items-center gap-2.5 text-gray-12 text-sm motion-reduce:animate-none"
                style={{ animationDelay: `${120 + i * 30}ms` }}
              >
                <Icon className="size-4 shrink-0 text-gray-11" />
                {label}
              </li>
            ))}
          </ul>
          <Button
            variant="primary"
            size="lg"
            className="w-full animate-upgrade-rise motion-reduce:animate-none"
            style={{ animationDelay: "200ms" }}
            onClick={() => setOpen(false)}
          >
            Done
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function CoinHeader({ back }: { back: ReactNode }) {
  return (
    <div className="relative flex h-32 items-center justify-center overflow-hidden pt-4">
      <div className="relative text-gray-12">
        <span className="block animate-mark-float motion-reduce:animate-none">
          <span className="block animate-coin-hop motion-reduce:animate-none">
            <SpinCoin
              front={
                <svg
                  viewBox="0 0 512 512"
                  fill="currentColor"
                  aria-label="Unkey"
                  role="img"
                  className="size-9 animate-mark-reveal [mask-image:linear-gradient(135deg,black_40%,transparent_60%)] [mask-position:0%_0%] [mask-size:300%_300%] motion-reduce:animate-none"
                >
                  <path d="M170.8 115V340.6H341.2L284.4 397H170.8C139.418 397 114 371.761 114 340.6V115H170.8Z" />
                  <path d="M398 284.2L341.2 340.6V115H398V284.2Z" />
                </svg>
              }
              back={back}
            />
          </span>
        </span>
      </div>
    </div>
  );
}
