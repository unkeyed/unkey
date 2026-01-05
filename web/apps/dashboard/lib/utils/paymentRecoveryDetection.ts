import Stripe from "stripe";

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
  async isRecoveryFromFailure(
    successEvent: Stripe.Event,
    invoiceId: string
  ): Promise<boolean> {
    try {
      // Extract the invoice from the success event
      const invoice = successEvent.data.object as Stripe.Invoice;
      
      // Get the customer ID for filtering events
      const customerId = typeof invoice.customer === 'string' 
        ? invoice.customer 
        : invoice.customer?.id;
      
      if (!customerId) {
        return false;
      }

      // Define the time window for checking recent failures (24 hours)
      const timeWindowHours = 24;
      const timeWindowSeconds = timeWindowHours * 60 * 60;
      const successTimestamp = successEvent.created;
      const earliestFailureTime = successTimestamp - timeWindowSeconds;

      // Retrieve recent events for this customer to look for payment failures
      const recentEvents = await this.stripe.events.list({
        type: 'invoice.payment_failed',
        created: {
          gte: earliestFailureTime,
          lte: successTimestamp
        },
        limit: 100 // Reasonable limit to check recent failures
      });

      // Check if any recent failure events are for the same invoice or customer
      const hasRecentFailure = recentEvents.data.some(failureEvent => {
        const failedInvoice = failureEvent.data.object as Stripe.Invoice;
        const failedCustomerId = typeof failedInvoice.customer === 'string'
          ? failedInvoice.customer
          : failedInvoice.customer?.id;

        // Check if it's the same invoice or same customer with recent failure
        return (
          failedInvoice.id === invoiceId || 
          (failedCustomerId === customerId && this.isRecentFailure(failureEvent, successTimestamp))
        );
      });

      // Additional check: examine the invoice's payment attempt history
      const hasMultipleAttempts = await this.checkInvoicePaymentAttempts(invoice);

      return hasRecentFailure || hasMultipleAttempts;
    } catch (error) {
      console.error('Error detecting payment recovery:', error);
      // Fail safely - if we can't determine, assume it's not a recovery
      return false;
    }
  }

  /**
   * Checks if a failure event is recent enough to be considered for recovery detection
   * 
   * @param failureEvent - The payment failure event
   * @param successTimestamp - Timestamp of the success event
   * @returns boolean - True if the failure is recent enough
   */
  private isRecentFailure(failureEvent: Stripe.Event, successTimestamp: number): boolean {
    // Consider failures within the last 24 hours as recent
    const maxFailureAge = 24 * 60 * 60; // 24 hours in seconds
    const failureAge = successTimestamp - failureEvent.created;
    
    return failureAge <= maxFailureAge && failureAge >= 0;
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
        const paymentIntentId = typeof invoice.payment_intent === 'string'
          ? invoice.payment_intent
          : invoice.payment_intent.id;

        // Retrieve charges for this payment intent
        const charges = await this.stripe.charges.list({
          payment_intent: paymentIntentId,
          limit: 10
        });
        
        // Check if there were multiple charges (indicating retry attempts)
        if (charges.data.length > 1) {
          return true;
        }

        // Check for failed charges followed by successful ones
        const hasFailedCharge = charges.data.some((charge: Stripe.Charge) => charge.status === 'failed');
        const hasSuccessfulCharge = charges.data.some((charge: Stripe.Charge) => charge.status === 'succeeded');
        
        return hasFailedCharge && hasSuccessfulCharge;
      }

      return false;
    } catch (error) {
      console.error('Error checking invoice payment attempts:', error);
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

      const customerId = typeof invoice.customer === 'string'
        ? invoice.customer
        : invoice.customer.id;

      // Extract customer information from the invoice
      let customerEmail = '';
      let customerName = '';
      
      if (typeof invoice.customer === 'object' && invoice.customer && !('deleted' in invoice.customer)) {
        const customer = invoice.customer as Stripe.Customer;
        customerEmail = customer.email || '';
        customerName = customer.name || '';
      }

      const subscriptionId = typeof invoice.subscription === 'string'
        ? invoice.subscription
        : invoice.subscription?.id;

      // Extract failure reason for payment_failed events (from event data only)
      let failureReason: string | undefined;
      if (event.type === 'invoice.payment_failed') {
        failureReason = 'Payment failed';
      }

      return {
        customerId,
        customerEmail,
        customerName,
        amount: invoice.amount_due || 0,
        currency: invoice.currency || 'usd',
        invoiceId: invoice.id,
        subscriptionId,
        failureReason,
        attemptCount: invoice.attempt_count || 1,
        eventTimestamp: event.created
      };
    } catch (error) {
      console.error('Error extracting payment context:', error);
      return null;
    }
  }

  /**
   * Performs temporal analysis to detect recent failure patterns
   * 
   * @param customerId - The Stripe customer ID
   * @param currentTimestamp - Current event timestamp
   * @returns Promise<boolean> - True if recent failure patterns detected
   */
  async analyzeRecentFailurePatterns(
    customerId: string, 
    currentTimestamp: number
  ): Promise<boolean> {
    try {
      // Look for payment failures in the last 7 days
      const lookbackDays = 7;
      const lookbackSeconds = lookbackDays * 24 * 60 * 60;
      const earliestTime = currentTimestamp - lookbackSeconds;

      // Get recent payment failure events for this customer
      const failureEvents = await this.stripe.events.list({
        type: 'invoice.payment_failed',
        created: {
          gte: earliestTime,
          lte: currentTimestamp
        },
        limit: 50
      });

      // Filter events for this specific customer
      const customerFailures = failureEvents.data.filter(event => {
        const invoice = event.data.object as Stripe.Invoice;
        const eventCustomerId = typeof invoice.customer === 'string'
          ? invoice.customer
          : invoice.customer?.id;
        return eventCustomerId === customerId;
      });

      // Analyze failure patterns
      if (customerFailures.length === 0) {
        return false;
      }

      // Check for multiple failures in recent period
      if (customerFailures.length >= 2) {
        return true;
      }

      // Check if the single failure was very recent (within last 6 hours)
      const recentFailureThreshold = 6 * 60 * 60; // 6 hours in seconds
      const mostRecentFailure = customerFailures[0];
      const timeSinceFailure = currentTimestamp - mostRecentFailure.created;

      return timeSinceFailure <= recentFailureThreshold;
    } catch (error) {
      console.error('Error analyzing failure patterns:', error);
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
  successEvent: Stripe.Event
): Promise<boolean> {
  const detector = createPaymentRecoveryDetector(stripe);
  const invoice = successEvent.data.object as Stripe.Invoice;
  
  return detector.isRecoveryFromFailure(successEvent, invoice.id);
}