import { StripeLinkLogo } from "@/components/ui/icons";
import type Stripe from "stripe";

type UserPaymentMethod = {
  paymentMethod: Stripe.PaymentMethod;
};

export function UserPaymentMethod({ paymentMethod }: Partial<UserPaymentMethod>) {
  if (!paymentMethod) {
    return <MissingPaymentMethod />;
  }

  if (paymentMethod.card) {
    return <CreditCard paymentMethod={paymentMethod} />;
  }

  if (paymentMethod.link) {
    return <StripeLink paymentMethod={paymentMethod} />;
  }

  return <FallbackPaymentMethod paymentMethod={paymentMethod} />;
}

const CreditCard: React.FC<UserPaymentMethod> = ({ paymentMethod }) => (
  <div className="aspect-[86/54] max-w-[320px] border border-gray-200 justify-between rounded-lg bg-gradient-to-tr from-gray-200/70 dark:from-black to-gray-100 dark:to-gray-900 dark:border dark:border-gray-800  shadow-lg p-8 ">
    <div className="mt-16 font-mono text-content whitespace-nowrap">
      •••• •••• •••• {paymentMethod.card?.last4}
    </div>
    <div className="mt-2 font-mono text-sm text-content-subtle">
      {paymentMethod.billing_details.name ?? "Anonymous"}
    </div>
    <div className="mt-1 font-mono text-xs text-content-subtle">
      Expires {paymentMethod.card?.exp_month.toLocaleString("en-US", { minimumIntegerDigits: 2 })}/
      {paymentMethod.card?.exp_year}
    </div>
  </div>
);

const StripeLink: React.FC<UserPaymentMethod> = ({ paymentMethod }) => (
  <div className="max-w-[320px] border border-gray-200 justify-between rounded-lg bg-gradient-to-tr from-gray-200/70 dark:from-black to-gray-100 dark:to-gray-900 dark:border dark:border-gray-800  shadow-lg p-8 ">
    <StripeLinkLogo className="rounded" />
    <div className="mt-6 font-mono text-content whitespace-nowrap">
      {paymentMethod.link?.email ?? "ja@example.com"}
    </div>
    <div className="mt-1 font-mono text-sm text-content-subtle">
      {paymentMethod.billing_details.name ?? "Anonymous"}
    </div>
  </div>
);

const FallbackPaymentMethod: React.FC<UserPaymentMethod> = ({ paymentMethod }) => (
  <div className="max-w-[320px] border border-gray-200 justify-between rounded-lg bg-gradient-to-tr from-gray-200/70 dark:from-black to-gray-100 dark:to-gray-900 dark:border dark:border-gray-800  shadow-lg p-8 ">
    <div className="font-mono text-sm text-content">
      {paymentMethod.billing_details.name ?? "Anonymous"}
    </div>
    <div className="z-50 mt-2 font-mono text-x text-content-subtle ">Saved payment method</div>
  </div>
);

const MissingPaymentMethod: React.FC = () => (
  <div className="relative aspect-[86/54] max-w-[320px] border border-gray-200 justify-between rounded-lg bg-gradient-to-tr from-gray-200/70 dark:from-black to-gray-100 dark:to-gray-900 dark:border dark:border-gray-800  shadow-lg p-8">
    <div className="z-50 mt-16 font-mono text-content whitespace-nowrap blur-sm">
      •••• •••• •••• ••••
    </div>
    <div className="z-50 mt-2 font-mono text-sm text-content-subtle ">No credit card on file</div>
    <div className="mt-1 font-mono text-xs text-content-subtle blur-sm">
      Expires {(new Date().getUTCMonth() - 1).toLocaleString("en-US", { minimumIntegerDigits: 2 })}/
      {new Date().getUTCFullYear()}
    </div>
  </div>
);
