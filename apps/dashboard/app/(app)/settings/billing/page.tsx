import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { Navbar } from "@/components/navbar";
import { PageContent } from "@/components/page-content";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { getOrgId } from "@/lib/auth";
import { clickhouse } from "@/lib/clickhouse";
import { type Workspace, db } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { cn } from "@/lib/utils";
import { type BillingTier, QUOTA, calculateTieredPrices } from "@unkey/billing";
import { Gear } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { Check, ExternalLink } from "lucide-react";
import Link from "next/link";
import { redirect } from "next/navigation";
import Stripe from "stripe";
import { navigation } from "../constants";
import { UserPaymentMethod } from "./user-payment-method";

export const revalidate = 0;

export default async function BillingPage() {
  const orgId = await getOrgId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
  });

  if (!workspace) {
    return redirect("/new");
  }

  return (
    <div>
      <Navbar>
        <Navbar.Breadcrumbs icon={<Gear />}>
          <Navbar.Breadcrumbs.Link href="/settings/billing" active>
            Settings
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
      </Navbar>
      <PageContent>
        <SubMenu navigation={navigation} segment="billing" />
        <div className="flex flex-col gap-8 lg:flex-row mt-8 ">
          <div className="w-full">
            {workspace.plan === "free" ? (
              <FreeUsage workspace={workspace} />
            ) : (
              <PaidUsage workspace={workspace} />
            )}
          </div>
          <Side workspace={workspace} />
        </div>
      </PageContent>
    </div>
  );
}

const FreeUsage: React.FC<{ workspace: Workspace }> = async ({ workspace }) => {
  const t = new Date();
  t.setUTCDate(1);
  t.setUTCHours(0, 0, 0, 0);

  const year = t.getUTCFullYear();
  const month = t.getUTCMonth() + 1;

  const usedVerifications = await clickhouse.billing.billableVerifications({
    workspaceId: workspace.id,
    year,
    month,
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Free Tier</CardTitle>
        <CardDescription>
          Current cycle:{" "}
          <span className="font-medium text-primary">
            {t.toLocaleString("en-US", { month: "long", year: "numeric" })}
          </span>{" "}
        </CardDescription>
      </CardHeader>

      <CardContent className="flex flex-col gap-8 md:flex-row">
        <ol className="flex flex-col w-2/3 space-y-6">
          <MeteredLineItem
            title="Verifications"
            tiers={[
              {
                firstUnit: 1,
                lastUnit: QUOTA.free.maxVerifications,
                centsPerUnit: null,
              },
            ]}
            used={usedVerifications}
          />
        </ol>
        <div className="w-1/3">
          <h4 className="font-medium">Upgrade your workspace</h4>

          <ul className="mt-2 space-y-1 text-sm text-content-subtle">
            <li className="flex items-center gap-2">
              <Check className="w-4 h-4" />
              150k Verifications
            </li>
            <li className="flex items-center gap-2">
              <Check className="w-4 h-4" />
              Unlimited team members
            </li>
            <li className="flex items-center gap-2">
              <Check className="w-4 h-4" />
              Priority support
            </li>
            <li className="flex items-center gap-2">
              <Check className="w-4 h-4" />
              90 days analytics retention
            </li>
          </ul>
        </div>
      </CardContent>
    </Card>
  );
};

const Side: React.FC<{ workspace: Workspace }> = async ({ workspace }) => {
  const env = stripeEnv();
  if (!env) {
    console.warn("No stripe env");
    return null;
  }

  const stripe = new Stripe(env.STRIPE_SECRET_KEY, {
    apiVersion: "2023-10-16",
    typescript: true,
  });

  let paymentMethod: Stripe.PaymentMethod | undefined = undefined;
  let invoices: Stripe.Invoice[] = [];
  let coupon: Stripe.Coupon | undefined = undefined;

  if (workspace.stripeCustomerId) {
    const [customer, paymentMethods, paid, open] = await Promise.all([
      stripe.customers.retrieve(workspace.stripeCustomerId),
      stripe.customers.listPaymentMethods(workspace.stripeCustomerId),
      stripe.invoices.list({
        customer: workspace.stripeCustomerId,
        limit: 3,
        status: "paid",
      }),
      stripe.invoices.list({
        customer: workspace.stripeCustomerId,
        limit: 3,
        status: "open",
      }),
    ]);

    invoices = [...open.data, ...paid.data].sort((a, b) => a.created - b.created);
    if (!customer.deleted) {
      coupon = customer.discount?.coupon;
    }
    if (paymentMethods && paymentMethods.data.length > 0) {
      paymentMethod = paymentMethods.data.at(0);
    }
  }

  return (
    <div className="w-full px-4 md:px-0 lg:w-2/5">
      <div className="flex flex-col items-center justify-center gap-8 md:flex-row lg:flex-col">
        <div className="flex flex-col w-full gap-8">
          <UserPaymentMethod paymentMethod={paymentMethod} />

          <div className="flex items-center gap-8">
            <Link href="/settings/billing/stripe" className="w-full">
              <Button className="whitespace-nowrap">
                {paymentMethod ? "Update Card" : "Add Credit Card"}
              </Button>
            </Link>
            <Link href="/settings/billing/plans">
              <Button className="whitespace-nowrap">Change Plan</Button>
            </Link>
          </div>
        </div>
        {coupon ? <Coupon coupon={coupon} /> : null}
        {invoices.length > 0 ? <Invoices invoices={invoices} /> : null}
      </div>
    </div>
  );
};

const PaidUsage: React.FC<{ workspace: Workspace }> = async ({ workspace }) => {
  const startOfMonth = new Date();
  startOfMonth.setUTCDate(1);
  startOfMonth.setUTCHours(0, 0, 0, 0);

  const year = startOfMonth.getUTCFullYear();
  const month = startOfMonth.getUTCMonth() + 1;

  const [usedVerifications, usedRatelimits] = await Promise.all([
    clickhouse.billing.billableVerifications({
      workspaceId: workspace.id,
      year,
      month,
    }),
    clickhouse.billing.billableRatelimits({
      workspaceId: workspace.id,
      year,
      month,
    }),
  ]);

  let currentPrice = 0;
  let estimatedTotalPrice = 0;
  if (workspace.subscriptions?.plan) {
    const cost = Number.parseFloat(workspace.subscriptions.plan.cents);
    currentPrice += cost;
    estimatedTotalPrice += cost; // does not scale
  }
  if (workspace.subscriptions?.support) {
    const cost = Number.parseFloat(workspace.subscriptions.support.cents);
    currentPrice += cost;
    estimatedTotalPrice += cost; // does not scale
  }

  if (workspace.subscriptions?.verifications) {
    const cost = calculateTieredPrices(
      workspace.subscriptions.verifications.tiers,
      usedVerifications,
    );
    if (cost.err) {
      return <div className="text-red-500">{cost.err.message}</div>;
    }
    currentPrice += cost.val.totalCentsEstimate;
    estimatedTotalPrice += forecastUsage(cost.val.totalCentsEstimate);
  }
  if (workspace.subscriptions?.ratelimits) {
    const cost = calculateTieredPrices(workspace.subscriptions.ratelimits.tiers, usedRatelimits);
    if (cost.err) {
      return <div className="text-red-500">{cost.err.message}</div>;
    }
    currentPrice += cost.val.totalCentsEstimate;
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>
          <span className="capitalize">{workspace.plan}</span> plan
        </CardTitle>
        <CardDescription>
          Current billing cycle:{" "}
          <span className="font-medium text-primary">
            {startOfMonth.toLocaleString("en-US", {
              month: "long",
              year: "numeric",
            })}
          </span>{" "}
        </CardDescription>
      </CardHeader>

      <CardContent>
        <ol className="flex flex-col space-y-6">
          {workspace.subscriptions?.plan ? (
            <LineItem title="Pro plan" cents={workspace.subscriptions.plan.cents} />
          ) : null}
          {workspace.subscriptions?.support ? (
            <LineItem title="Professional support" cents={workspace.subscriptions.support.cents} />
          ) : null}
          {workspace.subscriptions?.verifications ? (
            <MeteredLineItem
              displayPrice
              title="Verifications"
              tiers={workspace.subscriptions.verifications.tiers}
              used={usedVerifications}
              forecast
            />
          ) : null}
          {workspace.subscriptions?.ratelimits ? (
            <MeteredLineItem
              displayPrice
              title="Ratelimits"
              tiers={workspace.subscriptions.ratelimits.tiers}
              used={usedRatelimits}
              forecast
            />
          ) : null}
        </ol>
      </CardContent>
      <CardFooter className="flex flex-col gap-4">
        <div className="flex items-center justify-between w-full">
          <span className="text-sm font-semibold text-content">Current Total</span>
          <span className="text-sm font-semibold tabular-nums text-content">
            {formatCentsToDollar(currentPrice)}
          </span>
        </div>
        <div className="flex items-center justify-between w-full">
          <span className="text-xs text-content-subtle">Estimated by end of month</span>
          <span className="text-xs tabular-nums text-content-subtle">
            {formatCentsToDollar(estimatedTotalPrice)}
          </span>
        </div>
      </CardFooter>
    </Card>
  );
};

const LineItem: React.FC<{
  title: string;
  subtitle?: string;
  cents: string;
}> = (props) => (
  <div className="flex items-center justify-between">
    <div>
      <div className="font-semibold">
        <span className="capitalize">{props.title}</span>
      </div>
      <div className="text-sm text-secondary">{props.subtitle}</div>
    </div>
    <span className="text-sm font-semibold tabular-nums text-content">
      {formatCentsToDollar(Number.parseFloat(props.cents))}
    </span>
  </div>
);

function forecastUsage(currentUsage: number): number {
  const t = new Date();
  t.setUTCDate(1);
  t.setUTCHours(0, 0, 0, 0);

  const start = t.getTime();
  t.setUTCMonth(t.getUTCMonth() + 1);
  const end = t.getTime() - 1;

  const passed = (Date.now() - start) / (end - start);

  return currentUsage * (1 + 1 / passed);
}

const MeteredLineItem: React.FC<{
  displayPrice?: boolean;
  title: string;
  used: number;
  max?: number | null;
  tiers: BillingTier[];
  forecast?: boolean;
}> = (props) => {
  const firstTier = props.tiers.at(0);
  const included = firstTier?.centsPerUnit === null ? (firstTier.lastUnit ?? 0) : 0;

  const { val: price, err } = calculateTieredPrices(props.tiers, props.used);
  if (err) {
    return <div className="text-red-500">{err.message}</div>;
  }

  const forecast = forecastUsage(props.used);

  const currentTier = props.tiers.find((tier) => props.used >= tier.firstUnit);
  const max = props.max ?? Math.max(props.used, currentTier?.lastUnit ?? 0) * 1.2;

  return (
    <div className="flex items-center justify-between">
      <div
        className={cn("flex flex-col gap-1", {
          "w-2/3": props.displayPrice,
          "w-full": !props.displayPrice,
        })}
      >
        <div className="flex items-center justify-between">
          <span className="font-semibold capitalize text-content">{props.title}</span>
          {included ? (
            <span className="text-xs text-right text-content-subtle">
              {Intl.NumberFormat("en-US", { notation: "compact" }).format(included)} included
            </span>
          ) : null}
        </div>
        <div className="relative flex h-2 bg-gray-200 rounded-full dark:bg-gray-800">
          {props.tiers
            .filter((tier) => props.used >= tier.firstUnit)
            .map((tier, i) => (
              <Tooltip key={tier.firstUnit}>
                <TooltipTrigger
                  style={{
                    width: percentage(props.used - tier.firstUnit, max),
                  }}
                >
                  <div
                    className={cn("relative bg-primary hover:bg-brand duration-500 h-2", {
                      "opacity-100": i === 0,
                      "opacity-80": i === 1,
                      "opacity-60": i === 2,
                      "opacity-40": i === 3,
                      "opacity-20": i === 4,
                      "rounded-l-full": i === 0,
                    })}
                  >
                    <div className="absolute inset-y-0 right-0 w-px h-6 -mt-2 opacity-100 bg-gradient-to-t from-transparent via-gray-900 dark:via-gray-100 to-transparent" />
                  </div>
                </TooltipTrigger>
                <TooltipContent>
                  {tier.centsPerUnit ? (
                    <div className="flex flex-wrap items-baseline justify-between px-4 py-2 gap-x-4 gap-y-2 sm:px-6 xl:px-8">
                      <dt className="text-sm font-medium leading-6 text-content-subtle">
                        {" "}
                        {tier.centsPerUnit
                          ? formatCentsToDollar(Number.parseFloat(tier.centsPerUnit), 4)
                          : "free"}{" "}
                        per unit
                      </dt>

                      <dd className="flex-none w-full text-3xl font-medium leading-10 tracking-tight text-content">
                        {tier.firstUnit} - {tier.lastUnit ?? "∞"}
                      </dd>
                    </div>
                  ) : (
                    `${tier.lastUnit} included`
                  )}
                </TooltipContent>
              </Tooltip>
            ))}
          <div className="bg-gradient-to-r max-w-[4rem] w-full from-primary to-transparent opacity-50" />
        </div>
        <div className="flex items-center justify-between">
          <span className="text-xs text-content-subtle">
            Used {Intl.NumberFormat("en-US", { notation: "compact" }).format(props.used)}
          </span>
          {props.forecast ? (
            <span className="text-xs text-content-subtle">
              {Intl.NumberFormat("en-US", { notation: "compact" }).format(forecast)} forecasted
            </span>
          ) : null}
        </div>
      </div>
      {props.displayPrice ? (
        <span
          className={cn("tabular-nums text-sm", {
            "text-content font-semibold ": price.totalCentsEstimate > 0,
            "text-content-subtle": price.totalCentsEstimate === 0,
          })}
        >
          {formatCentsToDollar(price.totalCentsEstimate)}
        </span>
      ) : null}
    </div>
  );
};

function formatCentsToDollar(cents: number, decimals = 2): string {
  return Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 2,
    maximumFractionDigits: decimals,
  }).format(cents / 100);
}

function percentage(num: number, total: number): `${number}%` {
  if (total === 0) {
    return "0%";
  }
  return `${Math.min(100, (num / total) * 100)}%`;
}

const Coupon: React.FC<{ coupon: Stripe.Coupon }> = ({ coupon }) => (
  <div className="w-full p-8 border border-gray-200 rounded-lg dark:border-gray-800">
    <dt className="text-sm font-medium leading-6 text-content-subtle">Discount</dt>
    <dd className="flex-none w-full font-mono text-xl font-medium leading-10 tracking-tight text-content">
      {coupon.name}
    </dd>
  </div>
);

const Invoices: React.FC<{ invoices: Stripe.Invoice[] }> = ({ invoices }) => (
  <div className="w-full">
    <h4 className="font-medium">Invoices</h4>

    <ul className="divide-y divide-gray-200 dark:divide-gray-800">
      {invoices.map((invoice) => (
        <li key={invoice.id} className="flex items-center justify-between py-2 gap-x-6">
          <div>
            <span className="text-sm font-semibold tabular-nums text-content">
              {formatCentsToDollar(invoice.total)}
            </span>

            <p className="mt-1 text-xs leading-5 whitespace-nowrap text-content-subtle">
              {invoice.custom_fields?.find((f) => f.name === "Billing Period")?.value ??
                (invoice.due_date ? new Date(invoice.due_date * 1000).toDateString() : null)}
            </p>
          </div>

          <div className="flex flex-col items-end">
            <div className="flex items-center gap-1">
              {invoice.status !== "paid" ? (
                <div className="p-px border rounded-full border-alert/50">
                  <div className="w-2 h-2 rounded-full bg-alert" />
                </div>
              ) : null}
              <span className="text-sm capitalize text-content">{invoice.status}</span>
            </div>
            {invoice.hosted_invoice_url ? (
              <Link
                href={invoice.hosted_invoice_url}
                target="_blank"
                className="flex items-center mt-1 text-xs leading-5 duration-150 whitespace-nowrap text-content-subtle hover:underline hover:text-content"
              >
                View Invoice
                <ExternalLink className="inline-block w-3 h-3 ml-1" />
              </Link>
            ) : null}
          </div>
        </li>
      ))}
    </ul>
  </div>
);
