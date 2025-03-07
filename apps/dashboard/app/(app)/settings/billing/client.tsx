"use client";
import { SettingCard } from "@/components/settings-card";
import type { Workspace } from "@/lib/db";
import { formatNumber } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { Button, Empty } from "@unkey/ui";
import ms from "ms";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import type Stripe from "stripe";
import { Confirm } from "./components/confirmation";
import { Shell } from "./components/shell";
import { Usage } from "./components/usage";

type Props = {
  usage: {
    current: number;
    max: number;
  };
  workspace: Workspace;
  subscription?: {
    id: string;
    status: Stripe.Subscription.Status;
    trialUntil?: number;
  };
  currentProductId?: string;
  products: Array<{
    id: string;
    name: string;
    priceId: string;
    dollar: number;
    quota: {
      requestsPerMonth: number;
    };
  }>;
};

export const Client: React.FC<Props> = (props) => {
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

  // isNew means a workspace has not had a trial or paid plan yet and we should create one

  const isNew = !props.workspace.stripeCustomerId;

  const allowUpdate =
    props.subscription && ["active", "trialing"].includes(props.subscription.status);
  const allowCancel =
    props.subscription && ["active", "trialing"].includes(props.subscription.status);
  const isFreeTier =
    !props.subscription || !["active", "trialing"].includes(props.subscription.status);
  const selectedProductIndex = allowUpdate
    ? props.products.findIndex((p) => p.id === props.currentProductId)
    : -1;

  return (
    <Shell>
      {props.subscription ? (
        <SusbcriptionStatus
          status={props.subscription.status}
          trialUntil={props.subscription.trialUntil}
        />
      ) : null}

      {isFreeTier ? <FreeTierAlert /> : null}
      <Usage current={props.usage.current} max={props.usage.max} />

      <div className="w-full flex-col flex  ">
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
              <div className=" text-accent-12 font-medium w-4/12">{p.name}</div>
              <div className="w-4/12 flex justify-end items-center gap-1">
                <span className="text-accent-12 ">{formatNumber(p.quota.requestsPerMonth)}</span>
                <span className="text-gray-11 ">requests</span>
              </div>
              <div className="flex items-center justify-between gap-4 w-4/12">
                <div className="flex items-center justify-end gap-1 w-full">
                  <span className="font-medium  text-accent-12 ">${p.dollar}</span>
                  <span className="text-gray-11 ">/mo</span>
                </div>

                {props.subscription ? (
                  <Confirm
                    title={`${i > selectedProductIndex ? "Upgrade" : "Downgrade"} to ${p.name}`}
                    description={`Changing to ${
                      p.name
                    } updates your request quota to ${formatNumber(
                      p.quota.requestsPerMonth,
                    )} per month immediately.`}
                    onConfirm={async () =>
                      updateSubscription.mutateAsync({
                        oldProductId: props.currentProductId!,
                        newProductId: p.id,
                      })
                    }
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
                      p.quota.requestsPerMonth,
                    )} per month immediately.`}
                    onConfirm={async () => {
                      const res = await createSubscription.mutateAsync({ productId: p.id });

                      if (res?.url) {
                        router.push(res.url);
                      }
                    }}
                    fineprint={
                      isNew ? "After 14 days, the trial converts to a paid subscription." : ""
                    }
                    trigger={(onClick) => (
                      <Button variant="outline" disabled={isSelected} onClick={onClick}>
                        {isNew ? "Start 14 day trial" : "Upgrade"}
                      </Button>
                    )}
                  />
                )}
              </div>
            </div>
          );
        })}
      </div>
      <div className="w-full">
        {props.workspace.stripeCustomerId ? (
          <SettingCard
            title="Billing Portal"
            border={props.subscription && allowCancel ? "top" : "both"}
            description="Manage Payment methods and see your invoices."
          >
            <div className="w-full flex justify-end">
              <Button variant="outline" size="lg">
                <Link href="/settings/billing/stripe/portal">Open Portal</Link>
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
            <div className="w-full flex justify-end">
              <Confirm
                title="Cancel plan"
                description="Canceling your plan will immediately downgrade your workspace to the free tier. Prorated credits are applied to your account and you can resume the subscription any time."
                onConfirm={() => cancelSubscription.mutateAsync()}
                trigger={(onClick) => (
                  <Button variant="outline" color="danger" onClick={onClick}>
                    Cancel Plan
                  </Button>
                )}
              />
            </div>
          </SettingCard>
        ) : null}
      </div>
    </Shell>
  );
};

const FreeTierAlert: React.FC = () => {
  return (
    <Empty className="border border-gray-4 rounded-xl">
      <Empty.Title>You are on the Free tier.</Empty.Title>
      <Empty.Description>
        The Free tier includes 250k requests of free usage.
        <br />
        To unlock additional usage and add team members, upgrade to Pro.{" "}
        <Link href="https://unkey.com/pricing" target="_blank" className="text-info-11 underline">
          See Pricing
        </Link>
      </Empty.Description>
    </Empty>
  );
};

const SusbcriptionStatus: React.FC<{ status: Stripe.Subscription.Status; trialUntil?: number }> = (
  props,
) => {
  switch (props.status) {
    case "active":
      return null;
    case "trialing":
      if (!props.trialUntil) {
        return null;
      }
      return (
        <SettingCard
          title="Trial"
          description={
            <>
              Your trial ends in{" "}
              <span className="text-accent-12">
                {ms(props.trialUntil - Date.now(), { long: true })}
              </span>{" "}
              on{" "}
              <span className="text-accent-12">
                {new Date(props.trialUntil).toLocaleDateString()}
              </span>
              .
            </>
          }
          border="both"
          className="border-info-7 bg-info-3"
        />
      );

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
          <div className="w-full flex justify-end">
            <Button variant="primary" size="lg">
              <Link href="/settings/billing/stripe/portal">Open Portal</Link>
            </Button>
          </div>
        </SettingCard>
      );
    case "paused":
    case "canceled":
  }
  return null;
};
