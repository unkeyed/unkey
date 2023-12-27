import { EmptyPlaceholder } from "@/components/dashboard/empty-placeholder";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { getTenantId } from "@/lib/auth";
import { type Workspace, db } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { activeKeys, verifications } from "@/lib/tinybird";
import { cn } from "@/lib/utils";
import { Check, ExternalLink } from "lucide-react";
import Link from "next/link";
import { redirect } from "next/navigation";
import Stripe from "stripe";
import { ChangePlan } from "./change-plan";

export const revalidate = 0;

export default async function BillingPage() {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
  });
  if (!workspace) {
    return redirect("/new");
  }

  return (
    <div className="flex gap-8 flex-col lg:flex-row ">
      <div className="w-full">
        {workspace.stripeSubscriptionId === null ? (
          <FreeUsage workspace={workspace} />
        ) : (
          <ProUsage workspace={workspace} />
        )}
      </div>
      <Side workspace={workspace} />
    </div>
  );
}

const FreeUsage: React.FC<{ workspace: Workspace }> = async ({ workspace }) => {
  const t = new Date();
  t.setUTCDate(1);
  t.setUTCHours(0, 0, 0, 0);

  const year = t.getUTCFullYear();
  const month = t.getUTCMonth() + 1;

  const [usedActiveKeys, usedVerifications] = await Promise.all([
    activeKeys({
      workspaceId: workspace.id,
      year,
      month,
    }).then((res) => res.data.at(0)?.keys ?? 0),
    verifications({
      workspaceId: workspace.id,
      year,
      month,
    }).then((res) => res.data.at(0)?.success ?? 0),
  ]);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Free Tier</CardTitle>
        <CardDescription>
          Current cycle:{" "}
          <span className="text-primary font-medium">
            {t.toLocaleString("en-US", { month: "long", year: "numeric" })}
          </span>{" "}
        </CardDescription>
      </CardHeader>

      <CardContent className="flex flex-col md:flex-row gap-8">
        <ol className="flex flex-col space-y-6 w-2/3">
          <MeteredLineItem
            title="Active keys"
            tiers={[
              {
                up_to: 100,
                unit_amount: null,
                unit_amount_decimal: null,
                flat_amount: null,
                flat_amount_decimal: null,
              },
            ]}
            used={usedActiveKeys}
            cents={0}
          />
          <MeteredLineItem
            title="Verifications"
            tiers={[
              {
                up_to: 2500,
                unit_amount: null,
                unit_amount_decimal: null,
                flat_amount: null,
                flat_amount_decimal: null,
              },
            ]}
            used={usedVerifications}
            cents={0}
          />
        </ol>
        <div className="w-1/3">
          <h4 className="font-medium">Upgrade your workspace</h4>

          <ul className="mt-2 space-y-1 text-sm text-content-subtle">
            <li className="flex items-center gap-2">
              <Check className="w-4 h-4" />
              100+ Active keys
            </li>
            <li className="flex items-center gap-2">
              <Check className="w-4 h-4" />
              2500+ Verifications
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
    console.error("Missing stripe env");
    return null;
  }

  const stripe = new Stripe(env.STRIPE_SECRET_KEY, {
    apiVersion: "2022-11-15",
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
    <div className="px-4 md:px-0 w-full lg:w-2/5">
      <div className="flex flex-col md:flex-row lg:flex-col gap-8 items-center justify-center">
        <div className="flex flex-col gap-8 w-full">
          {paymentMethod?.card ? (
            <CreditCard paymentMethod={paymentMethod} />
          ) : (
            <MissingCreditCard />
          )}

          <div className="flex items-center gap-8">
            <Link href="/app/settings/billing/stripe" className="w-full">
              <Button variant="secondary" type="button" size="block" className="whitespace-nowrap">
                {paymentMethod ? "Update Card" : "Add Credit Card"}
              </Button>
            </Link>
            <ChangePlan
              workspace={workspace}
              trigger={
                <Button variant="secondary" type="button" size="block">
                  Change Plan
                </Button>
              }
            />
          </div>
        </div>
        {coupon ? <Coupon coupon={coupon} /> : null}
        {invoices.length > 0 ? <Invoices invoices={invoices} /> : null}
      </div>
    </div>
  );
};

const ProUsage: React.FC<{ workspace: Workspace }> = async ({ workspace }) => {
  const e = stripeEnv();
  if (!e) {
    return (
      <EmptyPlaceholder>
        <EmptyPlaceholder.Title>Stripe is not configured</EmptyPlaceholder.Title>
        <EmptyPlaceholder.Description>
          If you are selfhosting Unkey, you need to configure Stripe in your environment variables.
        </EmptyPlaceholder.Description>
      </EmptyPlaceholder>
    );
  }
  const stripe = new Stripe(e.STRIPE_SECRET_KEY, {
    apiVersion: "2022-11-15",
    typescript: true,
  });

  const subscription = await stripe.subscriptions.retrieve(workspace.stripeSubscriptionId!, {
    expand: ["items.data.price.product", "items.data.price.tiers"],
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Pro plan</CardTitle>
        <CardDescription>
          Current billing cycle:{" "}
          <span className="text-primary font-medium">
            {new Date(subscription.current_period_start * 1000).toLocaleString("en-US", {
              day: "2-digit",
              month: "long",
              year: "numeric",
            })}{" "}
            -{" "}
            {new Date(subscription.current_period_end * 1000).toLocaleString("en-US", {
              day: "2-digit",
              month: "long",
              year: "numeric",
            })}
          </span>{" "}
        </CardDescription>
      </CardHeader>

      <CardContent>
        <ol className="flex flex-col space-y-6">
          {subscription.items.data.map(async (item) => {
            // @ts-ignore Stripe knows nothing
            const productName = item.price.product.name;

            if (item.price.billing_scheme === "per_unit") {
              return <LineItem title={productName} cents={item.price.unit_amount ?? 0} />;
            } else if (item.price.billing_scheme === "tiered") {
              const usage = await stripe.subscriptionItems.listUsageRecordSummaries(item.id);

              return (
                <MeteredLineItem
                  displayPrice
                  title={productName}
                  used={usage.data.at(0)?.total_usage ?? 0}
                  tiers={item.price.tiers ?? []}
                  cents={item.price.unit_amount ?? 0}
                />
              );
            }
            return null;
          })}
        </ol>
      </CardContent>
      <CardFooter className="flex flex-col gap-4">
        <div className="flex items-center justify-between w-full">
          <span className="font-semibold text-content text-sm">Current Total</span>
          <span className="font-semibold tabular-nums text-content text-sm">
            {formatCentsToDollar(0)}
          </span>
        </div>
        <div className="flex items-center justify-between w-full">
          <span className="text-content-subtle text-xs">Estimated by end of month</span>
          <span className="tabular-nums text-content-subtle text-xs">{formatCentsToDollar(0)}</span>
        </div>
      </CardFooter>
    </Card>
  );
};

const LineItem: React.FC<{
  title: string;
  subtitle?: string;
  cents: number;
}> = (props) => (
  <div className="flex items-center justify-between">
    <div>
      <div className="font-semibold">
        <span className="capitalize">{props.title}</span>
      </div>
      <div className="text-sm text-secondary">{props.subtitle}</div>
    </div>
    <span className="font-semibold tabular-nums text-content text-sm">
      {formatCentsToDollar(props.cents)}
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
  tiers: Stripe.Price.Tier[];
  forecast?: boolean;
  cents: number;
}> = (props) => {
  const firstTier = props.tiers.at(0);
  const included = !firstTier?.unit_amount ? firstTier?.up_to ?? 0 : 0;

  const forecast = forecastUsage(props.used);

  const _freeTier = props.tiers.find(
    (t) =>
      !t.flat_amount && !t.unit_amount && !t.flat_amount_decimal && t.unit_amount_decimal === "0",
  );
  const currentTier = props.tiers.find((tier) => tier.up_to === null || props.used <= tier.up_to);

  return (
    <div className="flex items-center justify-between">
      <div
        className={cn("flex flex-col gap-1", {
          "w-2/3": props.displayPrice,
          "w-full": !props.displayPrice,
        })}
      >
        <div className="flex items-center justify-between">
          <span className="capitalize font-semibold text-content">{props.title}</span>
          {included ? (
            <span className="text-right text-xs text-content-subtle">
              {Intl.NumberFormat("en-US", { notation: "compact" }).format(included)} included
            </span>
          ) : null}
        </div>
        <div className="h-2 flex rounded-full bg-gray-200 dark:bg-gray-800 relative">
          <div
            style={{
              width: percentage(props.used, currentTier?.up_to ?? props.used),
            }}
          >
            <div className="relative bg-primary h-2 rounded-l-full">
              <div className="absolute opacity-100 right-0 inset-y-0 h-6 -mt-2 w-px bg-gradient-to-t from-transparent via-gray-900 dark:via-gray-100 to-transparent" />
            </div>
          </div>

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
            "text-content font-semibold ": props.cents > 0,
            "text-content-subtle": !props.cents || props.cents === 0,
          })}
        >
          {formatCentsToDollar(props.cents)}
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

const CreditCard: React.FC<{ paymentMethod: Stripe.PaymentMethod }> = ({ paymentMethod }) => (
  <div className="aspect-[86/54] max-w-[320px] border border-gray-200 dark:border-gray-800 justify-between rounded-lg bg-gradient-to-tr from-gray-200/70 dark:from-black to-gray-100 dark:to-gray-900 dark:border shadow-lg p-8 ">
    <div className="mt-16 font-mono text-content whitespace-nowrap">
      •••• •••• •••• {paymentMethod.card?.last4}
    </div>
    <div className="text-content-subtle font-mono text-sm mt-2">
      {paymentMethod.billing_details.name ?? "Anonymous"}
    </div>
    <div className="text-content-subtle text-xs font-mono mt-1">
      Expires {paymentMethod.card?.exp_month.toLocaleString("en-US", { minimumIntegerDigits: 2 })}/
      {paymentMethod.card?.exp_year}
    </div>
  </div>
);

const MissingCreditCard: React.FC = () => (
  <div className="relative aspect-[86/54] max-w-[320px] border border-gray-200 dark:border-gray-800 justify-between rounded-lg bg-gradient-to-tr from-gray-200/70 dark:from-black to-gray-100 dark:to-gray-900 dark:border dark:border-gray-800  shadow-lg p-8">
    <div className="z-50 mt-16 font-mono text-content whitespace-nowrap blur-sm">
      •••• •••• •••• ••••
    </div>
    <div className="z-50 text-content-subtle font-mono text-sm mt-2 ">No credit card on file</div>
    <div className="text-content-subtle text-xs font-mono mt-1 blur-sm">
      Expires {(new Date().getUTCMonth() - 1).toLocaleString("en-US", { minimumIntegerDigits: 2 })}/
      {new Date().getUTCFullYear()}
    </div>
  </div>
);

const Coupon: React.FC<{ coupon: Stripe.Coupon }> = ({ coupon }) => (
  <div className="w-full border border-gray-200 dark:border-gray-800 p-8 rounded-lg">
    <dt className="text-sm font-medium leading-6 text-content-subtle">Discount</dt>
    <dd className="w-full flex-none font-mono text-xl font-medium leading-10 tracking-tight text-content">
      {coupon.name}
    </dd>
  </div>
);

const Invoices: React.FC<{ invoices: Stripe.Invoice[] }> = ({ invoices }) => (
  <div className="w-full">
    <h4 className="font-medium">Invoices</h4>

    <ul className="divide-y divide-gray-200 dark:divide-gray-800">
      {invoices.map((invoice) => (
        <li key={invoice.id} className="flex items-center justify-between gap-x-6 py-2">
          <div>
            <span className="tabular-nums text-sm text-content font-semibold">
              {formatCentsToDollar(invoice.total)}
            </span>

            <p className="whitespace-nowrap mt-1 text-xs leading-5 text-content-subtle">
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
              <span className="text-sm text-content capitalize">{invoice.status}</span>
            </div>
            {invoice.hosted_invoice_url ? (
              <Link
                href={invoice.hosted_invoice_url}
                target="_blank"
                className="items-center flex whitespace-nowrap mt-1  leading-5 text-content-subtle hover:underline text-xs hover:text-content duration-150"
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
