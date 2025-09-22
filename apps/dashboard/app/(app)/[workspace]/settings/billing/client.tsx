"use client";
import type { Workspace } from "@/lib/db";
import { formatNumber } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { Button, Empty, SettingCard, toast } from "@unkey/ui";
import ms from "ms";
import Link from "next/link";
import { useRouter } from "next/navigation";
import type Stripe from "stripe";
import { WorkspaceNavbar } from "../workspace-navbar";
import { Confirm } from "./components/confirmation";
import { Shell } from "./components/shell";
import { Usage } from "./components/usage";

type Props = {
  hasPreviousSubscriptions: boolean;
  usage: {
    current: number;
    max: number;
  };
  workspace: Workspace;
  subscription?: {
    id: string;
    status: Stripe.Subscription.Status;
    cancelAt?: number;
  };
  currentProductId?: string;
  products: Array<{
    id: string;
    name: string;
    priceId: string;
    dollar: number;
    quotas: {
      requestsPerMonth: number;
    };
  }>;
};

const useBillingMutations = () => {
  const router = useRouter();

  const createSubscription = trpc.stripe.createSubscription.useMutation({
    onSuccess: () => {
      router.refresh();
      toast.info("Subscription started");
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });
  const updateSubscription = trpc.stripe.updateSubscription.useMutation({
    onSuccess: () => {
      router.refresh();
      toast.info("Plan changed");
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });
  const cancelSubscription = trpc.stripe.cancelSubscription.useMutation({
    onSuccess: () => {
      router.refresh();
      toast.info("Subscription cancelled");
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });
  const uncancelSubscription = trpc.stripe.uncancelSubscription.useMutation({
    onSuccess: () => {
      router.refresh();
      toast.info("Subscription resumed");
    },
    onError: (err) => {
      toast.error(err.message);
    },
  });

  return {
    createSubscription,
    updateSubscription,
    cancelSubscription,
    uncancelSubscription,
  };
};

export const Client: React.FC<Props> = (props) => {
  const mutations = useBillingMutations();
  const allowUpdate =
    props.subscription && ["active", "trialing"].includes(props.subscription.status);
  const allowCancel =
    props.subscription && props.subscription.status === "active" && !props.subscription.cancelAt;
  const isFreeTier = !props.subscription || props.subscription.status !== "active";
  const selectedProductIndex = allowUpdate
    ? props.products.findIndex((p) => p.id === props.currentProductId)
    : -1;

  return (
    <div>
      <WorkspaceNavbar activePage={{ href: "billing", text: "Billing" }} />
      <Shell>
        {props.subscription ? (
          <SubscriptionStatus workspaceId={props.workspace.id} status={props.subscription.status} />
        ) : null}

        <CancelAlert cancelAt={props.subscription?.cancelAt} />
        {isFreeTier ? <FreeTierAlert /> : null}
        <Usage current={props.usage.current} max={props.usage.max} />

        {props.workspace.stripeCustomerId ? (
          <div className="flex flex-col w-full">
            {props.products.map((p, i) => {
              const isSelected = selectedProductIndex === i;
              const isNextSelected = selectedProductIndex === i + 1;
              return (
                <div
                  key={p.id}
                  className={cn(
                    "text-sm border border-gray-4 px-6 py-3 w-full flex gap-6 justify-between items-center",
                    {
                      "rounded-t-xl": i === 0,
                      "border-t-0": i > 0 && !isSelected,
                      "border-b-0": isNextSelected,
                      "rounded-b-xl": i === props.products.length - 1,
                      "border-info-7 bg-info-3": isSelected,
                    },
                  )}
                >
                  <div className="w-4/12 font-medium text-accent-12">{p.name}</div>
                  <div className="flex items-center justify-end w-4/12 gap-1">
                    <span className="text-accent-12 ">
                      {formatNumber(p.quotas.requestsPerMonth)}
                    </span>
                    <span className="text-gray-11 ">requests</span>
                  </div>
                  <div className="flex items-center justify-between w-4/12 gap-4">
                    <div className="flex items-center justify-end w-full gap-1">
                      <span className="font-medium text-accent-12 ">${p.dollar}</span>
                      <span className="text-gray-11 ">/mo</span>
                    </div>

                    {props.subscription ? (
                      <Confirm
                        title={`${i > selectedProductIndex ? "Upgrade" : "Downgrade"} to ${p.name}`}
                        description={`Changing to ${
                          p.name
                        } updates your request quota to ${formatNumber(
                          p.quotas.requestsPerMonth,
                        )} per month immediately.`}
                        onConfirm={async () => {
                          if (!props.currentProductId) {
                            console.error(
                              "Cannot update subscription: currentProductId is missing",
                            );
                            toast.error(
                              "Unable to update subscription. Please refresh and try again.",
                            );
                            return;
                          }
                          await mutations.updateSubscription.mutateAsync({
                            oldProductId: props.currentProductId,
                            newProductId: p.id,
                          });
                        }}
                        trigger={(onClick) => (
                          <Button variant="outline" disabled={isSelected} onClick={onClick}>
                            Change
                          </Button>
                        )}
                      />
                    ) : (
                      <Confirm
                        title={`Upgrade to ${p.name}`}
                        description={`Changing to ${
                          p.name
                        } updates your request quota to ${formatNumber(
                          p.quotas.requestsPerMonth,
                        )} per month immediately.`}
                        onConfirm={() =>
                          mutations.createSubscription.mutateAsync({
                            productId: p.id,
                          })
                        }
                        trigger={(onClick) => (
                          <Button variant="outline" disabled={isSelected} onClick={onClick}>
                            Upgrade
                          </Button>
                        )}
                      />
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        ) : (
          <SettingCard
            title="Add payment method"
            border={props.subscription && allowCancel ? "top" : "both"}
            description="Before upgrading, you need to add a payment method."
            className="sm:w-full text-wrap w-full"
            contentWidth="w-full"
          >
            <div className="flex justify-end w-full">
              <Button variant="primary">
                <Link href={`/${props.workspace.id}/settings/billing/stripe/checkout`}>
                  Add payment method
                </Link>
              </Button>
            </div>
          </SettingCard>
        )}
        <div className="w-full">
          {props.workspace.stripeCustomerId ? (
            <SettingCard
              title="Billing Portal"
              border={props.subscription && allowCancel ? "top" : "both"}
              description="Manage Payment methods and see your invoices."
            >
              <div className="flex justify-end w-full">
                <Button variant="outline" size="lg">
                  <Link href={`/${props.workspace.id}/settings/billing/stripe/portal`}>
                    Open Portal
                  </Link>
                </Button>
              </div>
            </SettingCard>
          ) : null}

          {props.subscription && allowCancel ? (
            <SettingCard
              title="Cancel Subscription"
              description="Cancelling your subscription will downgrade your workspace to the free tier."
              border="bottom"
              className="border-t"
            >
              <div className="flex justify-end w-full">
                <Confirm
                  title="Cancel plan"
                  description="Canceling your plan will downgrade your workspace to the free tier at the end of the current period. You can resume your subscription until then."
                  onConfirm={() => mutations.cancelSubscription.mutateAsync()}
                  trigger={(onClick) => (
                    <Button variant="outline" color="danger" size="lg" onClick={onClick}>
                      Cancel Plan
                    </Button>
                  )}
                />
              </div>
            </SettingCard>
          ) : null}
        </div>
      </Shell>
    </div>
  );
};

const FreeTierAlert: React.FC = () => {
  return (
    <Empty className="border border-gray-4 rounded-xl">
      <Empty.Title>You are on the Free tier.</Empty.Title>
      <Empty.Description>
        The Free tier includes 150k requests of free usage.
        <br />
        To unlock additional usage and add team members, upgrade to Pro.{" "}
        <Link href="https://unkey.com/pricing" target="_blank" className="underline text-info-11">
          See Pricing
        </Link>
      </Empty.Description>
    </Empty>
  );
};

const CancelAlert: React.FC<{ cancelAt?: number }> = (props) => {
  const mutations = useBillingMutations();

  if (!props.cancelAt) {
    return null;
  }

  return (
    <SettingCard
      title="Cancellation scheduled"
      description={
        <p>
          Your subscription ends in
          <span className="text-accent-12"> {ms(props.cancelAt - Date.now(), { long: true })}</span>{" "}
          on <span className="text-accent-12">{new Date(props.cancelAt).toLocaleDateString()}</span>
          .
        </p>
      }
      border="both"
      className="border-warning-7 bg-warning-2"
    >
      <div className="flex justify-end w-full">
        <Button
          variant="primary"
          loading={mutations.uncancelSubscription.isLoading}
          disabled={mutations.uncancelSubscription.isLoading}
          onClick={() => mutations.uncancelSubscription.mutate()}
        >
          Resubscribe
        </Button>
      </div>
    </SettingCard>
  );
};
const SubscriptionStatus: React.FC<{
  status: Stripe.Subscription.Status;
  trialUntil?: number;
  workspaceId: string;
}> = (props) => {
  switch (props.status) {
    case "active":
      return null;

    case "incomplete":
    case "incomplete_expired":
    case "unpaid":
    case "past_due":
      return (
        <SettingCard
          title="Payment Required"
          description="There is a problem with your payment. Please resolve it."
          border="both"
          className="border-error-7 bg-error-3"
        >
          <div className="flex justify-end w-full">
            <Button variant="primary" size="lg">
              <Link href={`/${props.workspaceId}/settings/billing/stripe/portal`}>Open Portal</Link>
            </Button>
          </div>
        </SettingCard>
      );
    case "paused":
    case "canceled":
  }
  return null;
};
