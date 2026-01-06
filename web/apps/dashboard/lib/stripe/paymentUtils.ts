import type Stripe from "stripe";
import type { PreviousAttributes } from "./subscriptionUtils";

/**
 * Interface for payment context extracted from Stripe events
 */
interface PaymentContext {
  customerId: string;
  customerEmail: string;
  customerName: string;
  amount: number;
  currency: string;
  invoiceId: string;
  subscriptionId?: string;
  failureReason?: string;
  attemptCount?: number;
  eventTimestamp: number;
}

/**
 * Determines if a subscription update is due to payment recovery.
 * This happens when:
 * 1. Subscription status changed from past_due/unpaid to active
 * 2. Latest invoice changed (indicating successful payment processing)
 * 3. No manual changes to pricing, plan, or other subscription settings
 */
export async function isPaymentRecoveryUpdate(
  stripe: Stripe,
  sub: Stripe.Subscription,
  previousAttributes: PreviousAttributes | undefined,
  event: Stripe.Event,
): Promise<boolean> {
  if (!previousAttributes) {
    return false;
  }

  const changedKeys = Object.keys(previousAttributes);

  // Check if status changed from payment failure status to active
  const paymentFailureStatuses = ["past_due", "unpaid", "incomplete"];
  const statusRecovered =
    changedKeys.includes("status") &&
    paymentFailureStatuses.includes(previousAttributes.status || "") &&
    sub.status === "active";

  // Check if latest_invoice changed (indicates payment processing)
  const invoiceChanged = changedKeys.includes("latest_invoice");

  // Define keys that indicate manual changes (not payment-related)
  const manualChangeKeys = [
    "cancel_at_period_end",
    "collection_method",
    "plan",
    "quantity",
    "discount",
    "items", // pricing/plan changes
  ];

  // If any manual change keys are present, this is not a payment recovery update
  const hasManualChanges = manualChangeKeys.some((key) => changedKeys.includes(key));

  // Quick check: if status recovered to active without manual changes, it's a payment recovery
  if (statusRecovered && !hasManualChanges) {
    return true;
  }

  // If only invoice changed without manual changes, check if it was recently paid
  if (invoiceChanged && !hasManualChanges && sub.status === "active") {
    return await checkRecentPaymentSuccess(stripe, sub, event);
  }

  return false;
}

/**
 * Checks if there was a recent successful payment for the subscription.
 * Used to determine if a subscription update is due to payment recovery.
 */
export async function checkRecentPaymentSuccess(
  stripe: Stripe,
  sub: Stripe.Subscription,
  event: Stripe.Event,
): Promise<boolean> {
  try {
    // Instead of fetching many events, check the subscription's latest invoice directly
    if (!sub.latest_invoice) {
      return false;
    }

    const invoiceId =
      typeof sub.latest_invoice === "string" ? sub.latest_invoice : sub.latest_invoice.id;

    // Retrieve the latest invoice to check its payment status
    const invoice = await stripe.invoices.retrieve(invoiceId);

    // Check if the invoice was recently paid successfully
    if (invoice.status === "paid" && invoice.status_transitions?.paid_at) {
      // Consider it recent if paid within the last 2 hours
      const recentPaymentThreshold = 2 * 60 * 60; // 2 hours in seconds
      const timeSincePayment = event.created - invoice.status_transitions.paid_at;

      return timeSincePayment <= recentPaymentThreshold && timeSincePayment >= 0;
    }

    return false;
  } catch (error) {
    console.error("Error checking recent payment success:", {
      error:
        error instanceof Error
          ? {
              message: error.message,
              stack: error.stack,
              name: error.name,
            }
          : error,
      subscriptionId: sub.id,
      eventId: event.id,
    });
    // Fail safely - if we can't determine, assume it's not a recovery
    return false;
  }
}

/**
 * Stateless payment recovery detector that uses only Stripe webhook event data
 * to determine if a payment success follows a previous failure
 */
export class PaymentRecoveryDetector {
  private stripe: Stripe;

  constructor(stripe: Stripe) {
    this.stripe = stripe;
  }

  /**
   * Determines if a payment success follows a recent failure
   * Uses Stripe event metadata and timestamps for stateless detection
   *
   * @param successEvent - The invoice.payment_succeeded event
   * @param invoiceId - The invoice ID from the success event
   * @returns Promise<boolean> - True if this success follows a recent failure
   */
  async isRecoveryFromFailure(successEvent: Stripe.Event, invoiceId: string): Promise<boolean> {
    try {
      // Extract the invoice from the success event
      const invoice = successEvent.data.object as Stripe.Invoice;

      // Check if this is likely an upgrade/downgrade payment
      if (await this.isSubscriptionChangePayment(invoice)) {
        return false;
      }

      // Get the customer ID for filtering events
      const customerId =
        typeof invoice.customer === "string" ? invoice.customer : invoice.customer?.id;

      if (!customerId) {
        console.warn("Payment recovery detection: No customer ID found", {
          invoiceId,
          eventId: successEvent.id,
        });
        return false;
      }

      // Define the time window for checking recent failures (24 hours)
      const timeWindowHours = 24;
      const timeWindowSeconds = timeWindowHours * 60 * 60;
      const successTimestamp = successEvent.created;
      const earliestFailureTime = successTimestamp - timeWindowSeconds;

      let recentEvents: Stripe.ApiList<Stripe.Event>;
      try {
        // Retrieve recent events for this customer to look for payment failures
        recentEvents = await this.stripe.events.list({
          type: "invoice.payment_failed",
          created: {
            gte: earliestFailureTime,
            lte: successTimestamp,
          },
          limit: 100, // Reasonable limit to check recent failures
        });
      } catch (eventsError) {
        console.error("Failed to retrieve recent payment failure events:", {
          error: eventsError,
          customerId,
          invoiceId,
          eventId: successEvent.id,
        });
        // Fallback to checking invoice payment attempts only
        return await this.checkInvoicePaymentAttempts(invoice);
      }

      // Check if any recent failure events are for the SAME INVOICE ONLY
      // Don't consider failures from other invoices as recovery candidates
      const hasRecentFailure = recentEvents.data.some((failureEvent) => {
        try {
          const failedInvoice = failureEvent.data.object as Stripe.Invoice;

          // Only consider it a recovery if the EXACT SAME INVOICE failed and then succeeded
          // This prevents upgrade payments from being flagged as recoveries
          return failedInvoice.id === invoiceId;
        } catch (eventProcessingError) {
          console.warn("Error processing failure event during recovery detection:", {
            error: eventProcessingError,
            failureEventId: failureEvent.id,
          });
          return false;
        }
      });

      // Additional check: examine the invoice's payment attempt history
      let hasMultipleAttempts = false;
      try {
        hasMultipleAttempts = await this.checkInvoicePaymentAttempts(invoice);
      } catch (attemptsError) {
        console.error("Failed to check invoice payment attempts:", {
          error: attemptsError,
          invoiceId,
          eventId: successEvent.id,
        });
        // Continue with just the recent failure check
      }

      return hasRecentFailure || hasMultipleAttempts;
    } catch (error) {
      console.error("Error detecting payment recovery:", {
        error:
          error instanceof Error
            ? {
                message: error.message,
                stack: error.stack,
                name: error.name,
              }
            : error,
        invoiceId,
        eventId: successEvent.id,
      });
      // Fail safely - if we can't determine, assume it's not a recovery
      return false;
    }
  }

  /**
   * Determines if an invoice payment is likely due to a subscription change (upgrade/downgrade)
   * rather than a payment recovery scenario
   *
   * @param invoice - The Stripe invoice object
   * @returns Promise<boolean> - True if this appears to be a subscription change payment
   */
  private async isSubscriptionChangePayment(invoice: Stripe.Invoice): Promise<boolean> {
    try {
      // Check for subscription-related indicators
      if (!invoice.subscription) {
        return false;
      }

      const subscriptionId =
        typeof invoice.subscription === "string" ? invoice.subscription : invoice.subscription.id;

      // Check if this is a proration invoice, used in mid cycle upgrades
      if (invoice.lines?.data) {
        const hasProrationLines = invoice.lines.data.some(
          (line) =>
            line.proration === true || line.description?.toLowerCase().includes("proration"),
        );

        if (hasProrationLines) {
          return true;
        }
      }

      // Check if the invoice was created very recently relative to subscription billing cycle
      // Upgrade invoices are typically created immediately, not at billing cycle boundaries
      if (subscriptionId) {
        try {
          const subscription = await this.stripe.subscriptions.retrieve(subscriptionId);

          // If invoice was created significantly before the next billing cycle,
          // it's likely an upgrade/change
          const invoiceCreated = invoice.created;
          const nextBillingCycle = subscription.current_period_end;
          const timeUntilNextCycle = nextBillingCycle - invoiceCreated;

          // If more than 1 day until next billing cycle, a mid-cycle change
          const oneDayInSeconds = 24 * 60 * 60;
          if (timeUntilNextCycle > oneDayInSeconds) {
            console.info("Invoice created mid-cycle, likely subscription change", {
              invoiceId: invoice.id,
              subscriptionId,
              timeUntilNextCycle,
            });
            return true;
          }
        } catch (subscriptionError) {
          console.warn("Could not retrieve subscription for change detection:", {
            error: subscriptionError,
            subscriptionId,
            invoiceId: invoice.id,
          });
          // Continue with other checks
        }
      }

      return false;
    } catch (error) {
      console.error("Error checking if payment is subscription change:", {
        error:
          error instanceof Error
            ? {
                message: error.message,
                stack: error.stack,
                name: error.name,
              }
            : error,
        invoiceId: invoice.id,
      });
      // Fail safely - if we can't determine, assume it's not a subscription change
      return false;
    }
  }

  /**
   * Examines the invoice's payment attempt history to detect multiple attempts
   *
   * @param invoice - The Stripe invoice object
   * @returns Promise<boolean> - True if there were multiple payment attempts
   */
  private async checkInvoicePaymentAttempts(invoice: Stripe.Invoice): Promise<boolean> {
    try {
      // Check if the invoice has attempt_count metadata or multiple payment intents
      if (invoice.attempt_count && invoice.attempt_count > 1) {
        return true;
      }

      // If there's a payment intent, check its charges
      if (invoice.payment_intent) {
        const paymentIntentId =
          typeof invoice.payment_intent === "string"
            ? invoice.payment_intent
            : invoice.payment_intent.id;

        let charges: Stripe.ApiList<Stripe.Charge>;
        try {
          // Retrieve charges for this payment intent
          charges = await this.stripe.charges.list({
            payment_intent: paymentIntentId,
            limit: 10,
          });
        } catch (chargesError) {
          console.error("Error retrieving charges for payment intent:", {
            error: chargesError,
            paymentIntentId,
            invoiceId: invoice.id,
          });
          return false;
        }

        // Check if there were multiple charges (indicating retry attempts)
        if (charges.data.length > 1) {
          return true;
        }

        // Check for failed charges followed by successful ones
        const hasFailedCharge = charges.data.some(
          (charge: Stripe.Charge) => charge.status === "failed",
        );
        const hasSuccessfulCharge = charges.data.some(
          (charge: Stripe.Charge) => charge.status === "succeeded",
        );

        return hasFailedCharge && hasSuccessfulCharge;
      }

      return false;
    } catch (error) {
      console.error("Error checking invoice payment attempts:", {
        error:
          error instanceof Error
            ? {
                message: error.message,
                stack: error.stack,
                name: error.name,
              }
            : error,
        invoiceId: invoice.id,
      });
      return false;
    }
  }

  /**
   * Extracts payment context from a Stripe webhook event
   *
   * @param event - The Stripe webhook event
   * @returns PaymentContext | null - Extracted context or null if invalid
   */
  extractPaymentContext(event: Stripe.Event): PaymentContext | null {
    try {
      const invoice = event.data.object as Stripe.Invoice;

      if (!invoice.customer) {
        return null;
      }

      const customerId =
        typeof invoice.customer === "string" ? invoice.customer : invoice.customer.id;

      // Extract customer information from the invoice
      let customerEmail = "";
      let customerName = "";

      if (
        typeof invoice.customer === "object" &&
        invoice.customer &&
        !("deleted" in invoice.customer)
      ) {
        const customer = invoice.customer as Stripe.Customer;
        customerEmail = customer.email || "";
        customerName = customer.name || "";
      }

      const subscriptionId =
        typeof invoice.subscription === "string" ? invoice.subscription : invoice.subscription?.id;

      // Extract failure reason for payment_failed events (from event data only)
      let failureReason: string | undefined;
      if (event.type === "invoice.payment_failed") {
        failureReason = "Payment failed";
      }

      return {
        customerId,
        customerEmail,
        customerName,
        amount: invoice.amount_due || 0,
        currency: invoice.currency || "usd",
        invoiceId: invoice.id,
        subscriptionId,
        failureReason,
        attemptCount: invoice.attempt_count || 1,
        eventTimestamp: event.created,
      };
    } catch (error) {
      console.error("Error extracting payment context:", error);
      return null;
    }
  }

  /**
   * Performs temporal analysis to detect recent failure patterns for a specific invoice
   * Updated to be more precise about invoice-specific failures
   *
   * @param invoiceId - The specific invoice ID to check
   * @param customerId - The Stripe customer ID
   * @param currentTimestamp - Current event timestamp
   * @returns Promise<boolean> - True if recent failure patterns detected for this invoice
   */
  async analyzeRecentFailurePatterns(
    invoiceId: string,
    customerId: string,
    currentTimestamp: number,
  ): Promise<boolean> {
    try {
      // Look for payment failures in the last 7 days
      const lookbackDays = 7;
      const lookbackSeconds = lookbackDays * 24 * 60 * 60;
      const earliestTime = currentTimestamp - lookbackSeconds;

      let failureEvents: Stripe.ApiList<Stripe.Event>;
      try {
        // Get recent payment failure events for this customer
        failureEvents = await this.stripe.events.list({
          type: "invoice.payment_failed",
          created: {
            gte: earliestTime,
            lte: currentTimestamp,
          },
          limit: 50,
        });
      } catch (eventsError) {
        console.error("Error retrieving failure events for pattern analysis:", {
          error: eventsError,
          customerId,
          currentTimestamp,
        });
        return false;
      }

      // Filter events for this specific invoice only (not just customer)
      const invoiceFailures = failureEvents.data.filter((event) => {
        try {
          const invoice = event.data.object as Stripe.Invoice;
          // Only count failures for the exact same invoice
          return invoice.id === invoiceId;
        } catch (filterError) {
          console.warn("Error filtering failure event:", {
            error: filterError,
            eventId: event.id,
          });
          return false;
        }
      });

      // Only consider it a recovery if this specific invoice had previous failures
      return invoiceFailures.length > 0;
    } catch (error) {
      console.error("Error analyzing failure patterns:", {
        error:
          error instanceof Error
            ? {
                message: error.message,
                stack: error.stack,
                name: error.name,
              }
            : error,
        invoiceId,
        customerId,
        currentTimestamp,
      });
      return false;
    }
  }
}

/**
 * Factory function to create a PaymentRecoveryDetector instance
 *
 * @param stripe - Configured Stripe client
 * @returns PaymentRecoveryDetector instance
 */
export function createPaymentRecoveryDetector(stripe: Stripe): PaymentRecoveryDetector {
  return new PaymentRecoveryDetector(stripe);
}

/**
 * Utility function to determine if a payment success follows a failure
 * This is the main function that should be used in the webhook handler
 *
 * @param stripe - Configured Stripe client
 * @param successEvent - The invoice.payment_succeeded event
 * @returns Promise<boolean> - True if this success follows a recent failure
 */
export async function isPaymentRecovery(
  stripe: Stripe,
  successEvent: Stripe.Event,
): Promise<boolean> {
  const detector = createPaymentRecoveryDetector(stripe);
  const invoice = successEvent.data.object as Stripe.Invoice;

  return detector.isRecoveryFromFailure(successEvent, invoice.id);
}

export type { PaymentContext };
