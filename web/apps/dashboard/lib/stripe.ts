import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { stripeEnv } from "./env";

let stripeClient: Stripe | null = null;

/**
 * Get a singleton Stripe client instance.
 * Throws an error if Stripe is not configured.
 * The client is cached and reused across requests.
 */
export function getStripeClient(): Stripe {
  if (stripeClient) {
    return stripeClient;
  }

  const e = stripeEnv();
  if (!e) {
    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Stripe is not configured",
    });
  }

  stripeClient = new Stripe(e.STRIPE_SECRET_KEY, {
    apiVersion: "2023-10-16",
    typescript: true,
  });

  return stripeClient;
}
