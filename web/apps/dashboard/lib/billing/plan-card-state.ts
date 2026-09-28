import { formatDollars, formatPrice } from "@/lib/fmt";
import { DEPLOY_PLANS, type DeployPlan } from "@/lib/stripe/deployPlan";
import type { DeployPlanOption } from "@/lib/trpc/routers/stripe/getDeployPlans";

export type PlanPrice =
  | { type: "loading" }
  | { type: "amount"; cents: number; interval: string | null }
  | { type: "contact" };

export type PlanAction =
  | { type: "loading"; label: string }
  | { type: "unavailable"; label: string }
  | { type: "current"; label: string }
  | { type: "select"; label: string; option: DeployPlanOption };

export type CurrentPlan =
  | { status: "loading" }
  | { status: "error" }
  | { status: "none" }
  | { status: "plan"; plan: DeployPlan };

export function isComputeUpgrade(from: DeployPlan, to: DeployPlan): boolean {
  return DEPLOY_PLANS.indexOf(to) > DEPLOY_PLANS.indexOf(from);
}

export function currentPlanState(query: {
  isError: boolean;
  isLoading: boolean;
  plan: DeployPlan | null;
}): CurrentPlan {
  if (query.isError) {
    return { status: "error" };
  }
  if (query.isLoading) {
    return { status: "loading" };
  }
  return query.plan ? { status: "plan", plan: query.plan } : { status: "none" };
}

export type PlanCardState = {
  plan: DeployPlan;
  name: string;
  price: PlanPrice;
  action: PlanAction;
  warning: string | null;
};

function planLabel(plan: DeployPlan): string {
  return plan.charAt(0).toUpperCase() + plan.slice(1);
}

export function planName(plan: DeployPlan, options: DeployPlanOption[] | undefined): string {
  return options?.find((option) => option.plan === plan)?.name ?? planLabel(plan);
}

export function planCardStates({
  plans = DEPLOY_PLANS,
  options,
  current,
  usageCents,
}: {
  plans?: readonly DeployPlan[];
  options: DeployPlanOption[] | undefined;
  current: CurrentPlan;
  usageCents: number | null;
}): PlanCardState[] {
  const currentPlan = current.status === "plan" ? current.plan : null;
  const currentAmount = options?.find((option) => option.plan === currentPlan)?.amount ?? null;

  return plans.map((plan) => {
    const option = options?.find((o) => o.plan === plan);
    const name = option?.name ?? planLabel(plan);
    const isCurrent = plan === currentPlan;

    const price: PlanPrice =
      option === undefined
        ? { type: "loading" }
        : option.amount === null
          ? { type: "contact" }
          : { type: "amount", cents: option.amount, interval: option.interval };

    const isDowngrade =
      currentAmount !== null && option?.amount != null && option.amount < currentAmount;
    const verb =
      currentPlan === null
        ? "Choose"
        : currentAmount === null || option?.amount == null
          ? "Switch to"
          : isDowngrade
            ? "Downgrade to"
            : "Upgrade to";

    const warning =
      isDowngrade && option?.amount != null && usageCents !== null && usageCents > option.amount
        ? `Your usage this period (${formatPrice(usageCents)}) already exceeds the ${formatDollars(option.amount)} of monthly credits ${name} includes. This period keeps your current credits; from next period, usage at this level is billed as overage.`
        : null;

    const action: PlanAction = isCurrent
      ? { type: "current", label: "Current plan" }
      : current.status === "error"
        ? { type: "unavailable", label: `${verb} ${name}` }
        : option === undefined || current.status === "loading"
          ? { type: "loading", label: `${verb} ${name}` }
          : { type: "select", label: `${verb} ${name}`, option };

    return { plan, name, price, action, warning };
  });
}
