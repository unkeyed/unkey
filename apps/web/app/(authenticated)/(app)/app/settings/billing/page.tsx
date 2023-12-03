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
import { Workspace, db, eq, schema } from "@/lib/db";
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
    where: eq(schema.workspaces.tenantId, tenantId),
  });
  if (!workspace) {
    return redirect("/new");
  }

  return (
    <div className="flex gap-8 flex-col lg:flex-row ">
      <div className="w-full">
        {workspace.plan === "free" ? (
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
            tiers={[{ firstUnit: 1, lastUnit: 100, perUnit: 0 }]}
            used={usedActiveKeys}
            max={workspace.maxActiveKeys}
          />
          <MeteredLineItem
            title="Verifications"
            tiers={[{ firstUnit: 1, lastUnit: 2500, perUnit: 0 }]}
            used={usedVerifications}
            max={workspace.maxVerifications}
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
    return null;
  }

  const stripe = new Stripe(env.STRIPE_SECRET_KEY, {
    apiVersion: "2022-11-15",
    typescript: true,
  });

  let paymentMethod: Stripe.PaymentMethod | undefined = undefined;
  if (workspace.stripeCustomerId) {
    const paymentMethods = await stripe.customers.listPaymentMethods(workspace.stripeCustomerId);
    if (paymentMethods && paymentMethods.data.length > 0) {
      paymentMethod = paymentMethods.data.at(0);
    }
  }

  let invoices: Stripe.Invoice[] = [];
  if (workspace.stripeCustomerId) {
    const [paid, open] = await Promise.all([
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
        {invoices.length > 0 ? <Invoices invoices={invoices} /> : null}
      </div>
    </div>
  );
};

const ProUsage: React.FC<{ workspace: Workspace }> = async ({ workspace }) => {
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

  let totalPrice = 0;
  if (workspace.subscriptions?.plan) {
    totalPrice += workspace.subscriptions.plan.price;
  }
  if (workspace.subscriptions?.support) {
    totalPrice += workspace.subscriptions.support.price;
  }
  if (workspace.subscriptions?.activeKeys) {
    totalPrice += calculatePrice(workspace.subscriptions.activeKeys.tiers, usedActiveKeys);
  }
  if (workspace.subscriptions?.verifications) {
    totalPrice += calculatePrice(workspace.subscriptions.verifications.tiers, usedVerifications);
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Pro plan</CardTitle>
        <CardDescription>
          Current billing cycle:{" "}
          <span className="text-primary font-medium">
            {t.toLocaleString("en-US", { month: "long", year: "numeric" })}
          </span>{" "}
        </CardDescription>
      </CardHeader>

      <CardContent>
        <ol className="flex flex-col space-y-6">
          {workspace.subscriptions?.plan ? (
            <LineItem title="Pro plan" price={workspace.subscriptions.plan.price} />
          ) : null}
          {workspace.subscriptions?.support ? (
            <LineItem title="Professional support" price={workspace.subscriptions.support.price} />
          ) : null}
          {workspace.subscriptions?.activeKeys ? (
            <MeteredLineItem
              displayPrice
              title="Active keys"
              tiers={workspace.subscriptions.activeKeys.tiers}
              used={usedActiveKeys}
              max={workspace.maxActiveKeys}
            />
          ) : null}
          {workspace.subscriptions?.verifications ? (
            <MeteredLineItem
              displayPrice
              title="Verifications"
              tiers={workspace.subscriptions.verifications.tiers}
              used={usedVerifications}
              max={workspace.maxVerifications}
            />
          ) : null}
        </ol>
      </CardContent>
      <CardFooter className="flex items-center justify-between">
        <span className="font-semibold text-content text-sm">Current Total</span>
        <span className="font-semibold tabular-nums text-content text-sm">
          {dollar(totalPrice)}
        </span>
      </CardFooter>
    </Card>
  );
};

const LineItem: React.FC<{
  title: string;
  subtitle?: string;
  price: number; // in dollar
}> = (props) => (
  <div className="flex items-center justify-between">
    <div>
      <div className="font-semibold">
        <span className="capitalize">{props.title}</span>
      </div>
      <div className="text-sm text-secondary">{props.subtitle}</div>
    </div>
    <span className="font-semibold tabular-nums text-content text-sm">{dollar(props.price)}</span>
  </div>
);

const MeteredLineItem: React.FC<{
  displayPrice?: boolean;
  title: string;
  used: number;
  max?: number | null;
  tiers: {
    firstUnit: number;
    lastUnit: number | null; // null means unlimited
    perUnit: number; // $ 0.01
  }[];
}> = (props) => {
  const firstTier = props.tiers.at(0);
  const included = firstTier?.perUnit === 0 ? firstTier.lastUnit ?? 0 : 0;

  const price = calculatePrice(props.tiers, props.used);

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
        <div className="overflow-hidden rounded-full bg-gray-300 dark:bg-gray-800">
          <div
            className={cn("bg-primary h-2", {
              "bg-alert": props.max && props.used >= props.max,
            })}
            style={{ width: `${percentage(props.used, props.max ?? 0)}%` }}
          />
        </div>
        <span className="text-xs text-content-subtle">
          Used {Intl.NumberFormat("en-US", { notation: "compact" }).format(props.used)}
        </span>
      </div>
      {props.displayPrice ? (
        <span
          className={cn("tabular-nums text-sm", {
            "text-content font-semibold ": price > 0,
            "text-content-subtle": price === 0,
          })}
        >
          {dollar(price)}
        </span>
      ) : null}
    </div>
  );
};

function calculatePrice(
  tiers: {
    firstUnit: number;
    lastUnit: number | null; // null means unlimited
    perUnit: number; // $ 0.01
  }[],
  used: number,
): number {
  let price = 0;
  let u = used;
  for (const tier of tiers) {
    if (u <= 0) {
      break;
    }

    const quantity = tier.lastUnit === null ? u : Math.min(tier.lastUnit - tier.firstUnit + 1, u);
    u -= quantity;
    price += quantity * tier.perUnit;
  }
  return price;
}

function dollar(d: number): string {
  return Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(d);
}

function percentage(num: number, total: number): number {
  if (total === 0) {
    return 0;
  }
  return Math.min(100, (num / total) * 100);
}

const CreditCard: React.FC<{ paymentMethod: Stripe.PaymentMethod }> = ({ paymentMethod }) => (
  <div className="aspect-[86/54] max-w-[320px] border border-gray-200 dark:border-gray-800 justify-between rounded-lg bg-gradient-to-tr from-gray-200/70 dark:from-black to-gray-100 dark:to-gray-900 dark:border dark:border-gray-800  shadow-lg p-8 ">
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

const Invoices: React.FC<{ invoices: Stripe.Invoice[] }> = ({ invoices }) => (
  <div className="w-full">
    <h4 className="font-medium">Invoices</h4>

    <ul className="divide-y divide-gray-200 dark:divide-gray-800">
      {invoices.map((invoice) => (
        <li key={invoice.id} className="flex items-center justify-between gap-x-6 py-2">
          <div>
            <span className="tabular-nums text-sm text-content font-semibold">
              {dollar(invoice.total / 100)}
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
