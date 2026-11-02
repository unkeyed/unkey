import Stripe from "stripe";

export type StripeError = InstanceType<typeof Stripe.errors.StripeError>;

// Both signals are needed: some endpoints send `resource_missing` with a 400,
// others send a bare 404 with no code.
export function isStripeNotFound(error: StripeError): boolean {
  return error.statusCode === 404 || error.code === "resource_missing";
}

// `resource_missing` also fires for a restricted key and for a bad sub-resource
// named in the params. Only a miss on the path id (`param: "id"`, or none)
// means the target itself is gone.
function isMissingTarget(error: StripeError): boolean {
  if (error instanceof Stripe.errors.StripePermissionError) {
    return false;
  }
  if (
    error instanceof Stripe.errors.StripeInvalidRequestError &&
    error.param !== undefined &&
    error.param !== "id"
  ) {
    return false;
  }
  return isStripeNotFound(error);
}

export async function retrieveSubscriptionOrNull(
  stripe: Stripe,
  subscriptionId: string,
  params?: Stripe.SubscriptionRetrieveParams,
): Promise<Stripe.Subscription | null> {
  try {
    return await stripe.subscriptions.retrieve(subscriptionId, params);
  } catch (error) {
    if (error instanceof Stripe.errors.StripeError && isMissingTarget(error)) {
      return null;
    }
    throw error;
  }
}
