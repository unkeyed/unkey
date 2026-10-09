import { COMPUTE_SIGNUP_CREDIT_CENTS } from "@/lib/stripe/computeSignupCreditAmount";
import { describe, expect, it } from "vitest";
import {
  CREDITS_INFO,
  FIRST_INVOICE_CREDIT_NOTE,
  deployNotSubscribedSubtitle,
  firstInvoiceCents,
  planCreditsCopy,
} from "./compute-plan-copy";

describe("signup credit copy", () => {
  it("keeps the monthly credit line without the signup sentence", () => {
    expect(planCreditsCopy()).toBe(CREDITS_INFO);
    expect(planCreditsCopy()).toBe("Every plan includes monthly usage credit.");
    expect(planCreditsCopy()).not.toContain(
      "Each workspace gets one $5 credit on the first invoice.",
    );
    expect(deployNotSubscribedSubtitle()).toBe(
      "Run and scale your projects. Every plan includes usage credits equal to its fee.",
    );
    expect(deployNotSubscribedSubtitle()).not.toContain(
      "Each workspace gets one $5 credit on the first invoice.",
    );
  });

  it("takes $5 off the plan fee and floors the first invoice at $0", () => {
    expect(firstInvoiceCents(0)).toBe(0);
    expect(firstInvoiceCents(500)).toBe(0);
    expect(firstInvoiceCents(100)).toBe(0);
    expect(firstInvoiceCents(2500)).toBe(2000);
    expect(firstInvoiceCents(5000)).toBe(4500);
    expect(firstInvoiceCents(2500)).toBe(2500 - COMPUTE_SIGNUP_CREDIT_CENTS);
    expect(FIRST_INVOICE_CREDIT_NOTE).toBe("$5 on Unkey");
  });
});
