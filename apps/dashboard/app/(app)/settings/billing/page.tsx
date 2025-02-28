import { Navbar as SubMenu } from "@/components/dashboard/navbar";
import { Navbar } from "@/components/navbar";
import { PageContent } from "@/components/page-content";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { getTenantId } from "@/lib/auth";
import { clickhouse } from "@/lib/clickhouse";
import { type Workspace, db } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import { Gear } from "@unkey/icons";
import { Button, Empty } from "@unkey/ui";
import Link from "next/link";
import { redirect } from "next/navigation";
import Stripe from "stripe";
import { navigation } from "../constants";
import { ChangePlanButton } from "./change-plan";
import { UserPaymentMethod } from "./user-payment-method";
export const revalidate = 0;

export default async function BillingPage() {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAtM)),
  });

  if (!workspace) {
    return redirect("/new");
  }

  const e = stripeEnv();
  if (!e) {
    return (
      <Empty>
        <Empty.Title>Stripe is not configured</Empty.Title>
        <Empty.Description>
          If you are selfhosting Unkey, you need to configure Stripe in your environment variables.
        </Empty.Description>
      </Empty>
    );
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
            <Usage workspace={workspace} />
          </div>
          <Side workspace={workspace} />
        </div>
      </PageContent>
    </div>
  );
}

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
  const prices = await stripe.prices.list({
    product: "prod_RqmpxSwRAfrWbG",
  });

  let paymentMethod: Stripe.PaymentMethod | undefined = undefined;
  let coupon: Stripe.Coupon | undefined = undefined;

  let currentPriceId: string | undefined = undefined;

  if (workspace.stripeCustomerId) {
    const [customer, subscription, paymentMethods] = await Promise.all([
      stripe.customers.retrieve(workspace.stripeCustomerId),
      stripe.subscriptions.retrieve(workspace.stripeSubscriptionId!),
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
    if (!customer.deleted) {
      coupon = customer.discount?.coupon;
    }
    if (paymentMethods && paymentMethods.data.length > 0) {
      paymentMethod = paymentMethods.data.at(0);
    }
    if (subscription) {
      currentPriceId = subscription.items.data.at(0)?.price.id;
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
            <ChangePlanButton
              currentPriceId={currentPriceId}
              prices={prices.data
                .sort((a, b) => (a.unit_amount ?? 0) - (b.unit_amount ?? 0))
                .map((p) => ({
                  label: `${Intl.NumberFormat(undefined, { notation: "compact" }).format(
                    Number.parseInt(p.metadata.quota_requests ?? "0"),
                  )} requests for $${Intl.NumberFormat(undefined, { currency: "USD" }).format(
                    (p.unit_amount ?? 0) / 100,
                  )} /month`,
                  priceId: p.id,
                }))}
            />
          </div>
        </div>
        {coupon ? <Coupon coupon={coupon} /> : null}
      </div>
    </div>
  );
};

const Usage: React.FC<{ workspace: Workspace }> = async ({ workspace }) => {
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
        <MeteredLineItem
          displayPrice
          title="Requests"
          max={workspace.features.requestsQuota ?? 250_000}
          used={usedVerifications + usedRatelimits}
          forecast
        />
      </CardContent>
    </Card>
  );
};

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
  max: number;
  forecast?: boolean;
}> = (props) => {
  const forecast = forecastUsage(props.used);

  return (
    <div className="flex items-center justify-between">
      <div className="flex flex-col gap-1 w-full">
        <div className="flex items-center justify-between">
          <span className="font-semibold capitalize text-content">{props.title}</span>
          <span className="text-xs text-right text-content-subtle">
            {Intl.NumberFormat("en-US", { notation: "compact" }).format(props.max)}
          </span>
        </div>
        <div className="relative flex h-2 bg-gray-200 rounded-full dark:bg-gray-800">
          <div className="relative bg-primary hover:bg-brand duration-500 h-2 rounded-full">
            <div className="absolute inset-y-0 right-0 w-px h-6 -mt-2 opacity-100 bg-gradient-to-t from-transparent via-gray-900 dark:via-gray-100 to-transparent" />
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
    </div>
  );
};

const Coupon: React.FC<{ coupon: Stripe.Coupon }> = ({ coupon }) => (
  <div className="w-full p-8 border border-gray-200 rounded-lg dark:border-gray-800">
    <dt className="text-sm font-medium leading-6 text-content-subtle">Discount</dt>
    <dd className="flex-none w-full font-mono text-xl font-medium leading-10 tracking-tight text-content">
      {coupon.name}
    </dd>
  </div>
);
