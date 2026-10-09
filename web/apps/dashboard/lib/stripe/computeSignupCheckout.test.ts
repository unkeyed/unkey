import type Stripe from "stripe";
import { describe, expect, it, vi } from "vitest";

vi.mock("@/lib/db", () => ({
  db: { query: {} },
}));

vi.mock("@/lib/env", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/env")>();
  return { ...actual, stripeEnv: vi.fn() };
});

import {
  ComputeSignupCheckoutError,
  deployCardSetupSuccessUrl,
  deployCheckoutCustomerId,
  deployCheckoutSubmitMessage,
  prepareDeployCheckoutCredit,
} from "./computeSignupCheckout";
import { ComputeSignupCreditRetryError } from "./computeSignupCredit";
import type { ComputeSignupCreditClaim } from "./computeSignupCreditClaims";

const NOW_MS = 1_700_000_000_000;

function claim(overrides: Partial<ComputeSignupCreditClaim> = {}): ComputeSignupCreditClaim {
  return {
    pk: 1,
    cardFingerprint: "fpOk",
    workspaceId: "ws_1",
    stripeCustomerId: "cus_1",
    stripeBalanceTransactionId: null,
    workosUserId: "user_payer",
    attemptId: "attempt-1",
    createdAt: NOW_MS,
    ...overrides,
  };
}

class MemoryStore {
  rows: ComputeSignupCreditClaim[] = [];
  private nextPk = 1;

  async findByWorkspace(workspaceId: string) {
    return this.rows.find((row) => row.workspaceId === workspaceId) ?? null;
  }

  async findByFingerprint(fingerprint: string) {
    return this.rows.find((row) => row.cardFingerprint === fingerprint) ?? null;
  }

  async findByUser(workosUserId: string) {
    return this.rows.find((row) => row.workosUserId === workosUserId) ?? null;
  }

  async existsForWorkspaceOrUser(workspaceId: string, workosUserId: string) {
    return this.rows.some(
      (row) => row.workspaceId === workspaceId || row.workosUserId === workosUserId,
    );
  }

  async insert(row: {
    cardFingerprint: string;
    workspaceId: string;
    stripeCustomerId: string;
    workosUserId: string;
    createdAt: number;
    attemptId: string;
  }) {
    if (
      this.rows.some(
        (existing) =>
          existing.workspaceId === row.workspaceId ||
          existing.cardFingerprint === row.cardFingerprint ||
          existing.workosUserId === row.workosUserId,
      )
    ) {
      return "duplicate" as const;
    }
    this.rows.push({
      ...row,
      pk: this.nextPk,
      stripeBalanceTransactionId: null,
    });
    this.nextPk += 1;
    return "inserted" as const;
  }

  async claimAttempt(attempt: {
    pk: number;
    seenAttemptId: string;
    attemptId: string;
    createdAt: number;
    workspaceId: string;
    cardFingerprint: string;
    workosUserId: string;
    stripeCustomerId: string;
  }) {
    const row = this.rows.find((existing) => existing.pk === attempt.pk);
    if (!row || row.attemptId !== attempt.seenAttemptId || row.stripeBalanceTransactionId) {
      return false;
    }
    row.attemptId = attempt.attemptId;
    row.createdAt = attempt.createdAt;
    row.workspaceId = attempt.workspaceId;
    row.cardFingerprint = attempt.cardFingerprint;
    row.workosUserId = attempt.workosUserId;
    row.stripeCustomerId = attempt.stripeCustomerId;
    return true;
  }

  async setTransactionId(input: {
    workspaceId: string;
    cardFingerprint: string;
    workosUserId: string;
    attemptId: string;
    transactionId: string;
  }) {
    const row = this.rows.find(
      (existing) =>
        existing.workspaceId === input.workspaceId &&
        existing.cardFingerprint === input.cardFingerprint &&
        existing.workosUserId === input.workosUserId &&
        existing.attemptId === input.attemptId &&
        existing.stripeBalanceTransactionId === null,
    );
    if (!row) {
      return 0;
    }
    row.stripeBalanceTransactionId = input.transactionId;
    return 1;
  }

  async deleteAttempt() {
    return undefined;
  }
}

type Card = {
  id: string;
  fingerprint: string | null;
  funding: string | null;
  wallet: { type: string } | null;
};

function stripeStub(input?: {
  currency?: string | null;
  cards?: Card[];
  defaultPaymentMethodId?: string | null;
  setupSession?: Stripe.Checkout.Session;
  balanceTransactions?: Array<{ id: string; metadata: Stripe.Metadata | null }>;
  createError?: Error;
}) {
  const creates: Array<Record<string, unknown>> = [];
  const cards = input?.cards ?? [];
  const stripe = {
    customers: {
      retrieve: async () => ({
        id: "cus_1",
        deleted: undefined,
        currency: input?.currency === undefined ? "usd" : input.currency,
        invoice_settings: {
          default_payment_method: input?.defaultPaymentMethodId ?? null,
        },
      }),
      listBalanceTransactions: () => ({
        async *[Symbol.asyncIterator]() {
          for (const transaction of input?.balanceTransactions ?? []) {
            yield transaction;
          }
        },
      }),
      createBalanceTransaction: async (
        customerId: string,
        params: Record<string, unknown>,
        options?: { idempotencyKey?: string },
      ) => {
        if (input?.createError) {
          throw input.createError;
        }
        creates.push({ customerId, params, options });
        return {
          id: "cbtxn_1",
          metadata: params.metadata,
        };
      },
    },
    paymentMethods: {
      list: () => {
        const methods = cards.map((card) => ({
          id: card.id,
          type: "card" as const,
          card: {
            fingerprint: card.fingerprint,
            funding: card.funding,
            wallet: card.wallet,
          },
        }));
        return {
          data: methods.slice(0, 10),
          has_more: methods.length > 10,
          async *[Symbol.asyncIterator]() {
            for (const method of methods) {
              yield method;
            }
          },
        };
      },
      retrieve: async (id: string) => {
        const card = cards.find((candidate) => candidate.id === id);
        if (!card) {
          throw new Error(`missing payment method ${id}`);
        }
        return {
          id: card.id,
          type: "card",
          card: {
            fingerprint: card.fingerprint,
            funding: card.funding,
            wallet: card.wallet,
          },
        };
      },
    },
    checkout: {
      sessions: {
        retrieve: async () => {
          if (!input?.setupSession) {
            throw new Error("missing setup session");
          }
          return input.setupSession;
        },
      },
    },
  };
  return { stripe: stripe as unknown as Stripe, creates };
}

function setupSession(
  card: Card,
  customerId = "cus_new",
  intentUserId: string | null = "user_payer",
): Stripe.Checkout.Session {
  return {
    id: "cs_setup",
    mode: "setup",
    status: "complete",
    client_reference_id: "ws_1",
    customer: customerId,
    setup_intent: {
      id: "seti_1",
      metadata: intentUserId ? { workos_user_id: intentUserId } : {},
      payment_method: {
        id: card.id,
        type: "card",
        card: {
          fingerprint: card.fingerprint,
          funding: card.funding,
          wallet: card.wallet,
        },
      },
    },
  } as unknown as Stripe.Checkout.Session;
}

const base = {
  workspaceId: "ws_1",
  workosUserId: "user_payer",
  nowMs: NOW_MS,
};

describe("deployCheckoutCustomerId", () => {
  it("reuses the setup session customer instead of a clock customer minted on return", () => {
    expect(
      deployCheckoutCustomerId({
        devClockedCustomerId: "cus_clock",
        setupSessionId: "cs_setup",
      }),
    ).toBeUndefined();
    expect(
      deployCheckoutCustomerId({
        existingCustomerId: "cus_saved",
        devClockedCustomerId: "cus_clock",
        setupSessionId: "cs_setup",
      }),
    ).toBe("cus_saved");
    expect(
      deployCheckoutCustomerId({
        devClockedCustomerId: "cus_clock",
      }),
    ).toBe("cus_clock");
  });
});

describe("deployCheckoutSubmitMessage", () => {
  it("keeps the usage-credit line without the signup sentence", () => {
    const usage =
      "Your plan fee is matched by usage credits: $20.00 each month, and a prorated first charge is matched by the same amount in credits.";
    expect(
      deployCheckoutSubmitMessage({
        announceSignupCredit: true,
        signupCredit: "$5.00",
        planFee: "$20.00",
      }),
    ).toBe(usage);
    expect(
      deployCheckoutSubmitMessage({
        announceSignupCredit: false,
        signupCredit: "$5.00",
        planFee: "$20.00",
      }),
    ).toBe(usage);
  });
});

describe("deployCardSetupSuccessUrl", () => {
  it("returns to Compute checkout with the plan and the setup session placeholder", () => {
    const url = deployCardSetupSuccessUrl({
      baseUrl: "https://app.test",
      workspaceSlug: "acme",
      plan: "pro",
      from: "billing",
      returnTo: "/acme/projects",
    });

    expect(url).toBe(
      "https://app.test/acme/stripe/checkout?intent=deploy&plan=pro&from=billing&returnTo=/acme/projects&setup_session_id={CHECKOUT_SESSION_ID}",
    );
  });
});

describe("prepareDeployCheckoutCredit", () => {
  it("skips the card step when the workspace already claimed the credit", async () => {
    const store = new MemoryStore();
    store.rows.push(
      claim({
        stripeCustomerId: "cus_credited",
        stripeBalanceTransactionId: "cbtxn_done",
      }),
    );
    const { stripe, creates } = stripeStub();

    const result = await prepareDeployCheckoutCredit(stripe, {
      ...base,
      customerId: "cus_other",
      store,
    });

    expect(result).toEqual({
      step: "subscribe",
      customerId: "cus_credited",
      credit: null,
      announceSignupCredit: false,
    });
    expect(creates).toEqual([]);
    if (result.step === "subscribe") {
      expect(
        deployCheckoutSubmitMessage({
          announceSignupCredit: result.announceSignupCredit,
          signupCredit: "$5.00",
          planFee: "$20.00",
        }),
      ).toBe(
        "Your plan fee is matched by usage credits: $20.00 each month, and a prorated first charge is matched by the same amount in credits.",
      );
    }
  });

  it("collects a card when the workspace has no customer", async () => {
    const { stripe } = stripeStub();

    await expect(
      prepareDeployCheckoutCredit(stripe, { ...base, store: new MemoryStore() }),
    ).resolves.toEqual({ step: "collect_card" });
  });

  it("grants from an eligible saved card before subscribe", async () => {
    const store = new MemoryStore();
    const { stripe, creates } = stripeStub({
      defaultPaymentMethodId: "pm_credit",
      cards: [
        {
          id: "pm_prepaid",
          fingerprint: "fpPre",
          funding: "prepaid",
          wallet: null,
        },
        {
          id: "pm_credit",
          fingerprint: "fpOk",
          funding: "credit",
          wallet: null,
        },
      ],
    });

    const result = await prepareDeployCheckoutCredit(stripe, {
      ...base,
      customerId: "cus_1",
      store,
    });

    expect(result.step).toBe("subscribe");
    if (result.step === "subscribe") {
      expect(result.customerId).toBe("cus_1");
      expect(result.announceSignupCredit).toBe(true);
      expect(
        deployCheckoutSubmitMessage({
          announceSignupCredit: result.announceSignupCredit,
          signupCredit: "$5.00",
          planFee: "$20.00",
        }),
      ).toBe(
        "Your plan fee is matched by usage credits: $20.00 each month, and a prorated first charge is matched by the same amount in credits.",
      );
      expect(result.credit).toEqual({
        granted: true,
        transactionId: "cbtxn_1",
        amountCents: 500,
      });
    }
    expect(creates).toEqual([
      expect.objectContaining({
        customerId: "cus_1",
        params: expect.objectContaining({
          amount: -500,
          currency: "usd",
          metadata: expect.objectContaining({ card_fingerprint: "fpOk" }),
        }),
        options: { idempotencyKey: "compute-signup-credit:ws_1" },
      }),
    ]);
  });

  it("grants from a prepaid or wallet saved card", async () => {
    const { stripe, creates } = stripeStub({
      cards: [
        { id: "pm_pre", fingerprint: "fpPre", funding: "prepaid", wallet: null },
        {
          id: "pm_wallet",
          fingerprint: "fpWallet",
          funding: "credit",
          wallet: { type: "apple_pay" },
        },
      ],
    });

    const result = await prepareDeployCheckoutCredit(stripe, {
      ...base,
      customerId: "cus_1",
      store: new MemoryStore(),
    });

    expect(result).toMatchObject({
      step: "subscribe",
      announceSignupCredit: true,
      credit: { granted: true, amountCents: 500 },
    });
    expect(creates[0]?.params).toMatchObject({
      metadata: expect.objectContaining({ card_fingerprint: "fpPre" }),
    });
  });

  it("grants the returned card, then subscribes on that customer", async () => {
    const store = new MemoryStore();
    const { stripe, creates } = stripeStub({
      setupSession: setupSession({
        id: "pm_new",
        fingerprint: "fpNew",
        funding: "debit",
        wallet: null,
      }),
    });

    const result = await prepareDeployCheckoutCredit(stripe, {
      ...base,
      setupSessionId: "cs_setup",
      store,
    });

    expect(result).toMatchObject({
      step: "subscribe",
      customerId: "cus_new",
      announceSignupCredit: true,
      credit: { granted: true, amountCents: 500 },
    });
    expect(creates[0]?.params).toMatchObject({ amount: -500 });
  });

  it("grants the returned prepaid card", async () => {
    const { stripe, creates } = stripeStub({
      setupSession: setupSession({
        id: "pm_pre",
        fingerprint: "fpPre",
        funding: "prepaid",
        wallet: null,
      }),
    });

    const result = await prepareDeployCheckoutCredit(stripe, {
      ...base,
      setupSessionId: "cs_setup",
      store: new MemoryStore(),
    });

    expect(result).toMatchObject({
      step: "subscribe",
      customerId: "cus_new",
      announceSignupCredit: true,
      credit: { granted: true, amountCents: 500 },
    });
    expect(creates[0]?.params).toMatchObject({ amount: -500, currency: "usd" });
  });

  it("subscribes without a second credit when the card fingerprint is already claimed", async () => {
    const store = new MemoryStore();
    store.rows.push(
      claim({
        pk: 7,
        workspaceId: "ws_other",
        stripeCustomerId: "cus_other",
        cardFingerprint: "fpUsed",
        workosUserId: "user_other",
        stripeBalanceTransactionId: "cbtxn_used",
      }),
    );
    const { stripe, creates } = stripeStub({
      setupSession: setupSession({
        id: "pm_used",
        fingerprint: "fpUsed",
        funding: "credit",
        wallet: null,
      }),
    });

    const result = await prepareDeployCheckoutCredit(stripe, {
      ...base,
      setupSessionId: "cs_setup",
      store,
    });

    expect(result).toEqual({
      step: "subscribe",
      customerId: "cus_new",
      announceSignupCredit: false,
      credit: { granted: false, reason: "card fingerprint already used" },
    });
    expect(creates).toEqual([]);
  });

  it("grants a usd balance transaction when the customer currency is eur", async () => {
    const { stripe, creates } = stripeStub({
      currency: "eur",
      setupSession: setupSession(
        { id: "pm_eur", fingerprint: "fpEur", funding: "credit", wallet: null },
        "cus_eur",
      ),
    });

    const result = await prepareDeployCheckoutCredit(stripe, {
      ...base,
      customerId: "cus_eur",
      setupSessionId: "cs_setup",
      store: new MemoryStore(),
    });

    expect(result).toMatchObject({
      step: "subscribe",
      customerId: "cus_eur",
      announceSignupCredit: true,
      credit: { granted: true, amountCents: 500 },
    });
    expect(creates[0]?.params).toMatchObject({ amount: -500, currency: "usd" });
  });

  it("rejects a setup session for another workspace", async () => {
    const session = setupSession({
      id: "pm_new",
      fingerprint: "fpNew",
      funding: "credit",
      wallet: null,
    });
    session.client_reference_id = "ws_other";
    const { stripe } = stripeStub({ setupSession: session });

    await expect(
      prepareDeployCheckoutCredit(stripe, {
        ...base,
        setupSessionId: "cs_setup",
        store: new MemoryStore(),
      }),
    ).rejects.toBeInstanceOf(ComputeSignupCheckoutError);
  });

  it("rejects a setup session for a different customer", async () => {
    const { stripe } = stripeStub({
      setupSession: setupSession({
        id: "pm_new",
        fingerprint: "fpNew",
        funding: "credit",
        wallet: null,
      }),
    });

    await expect(
      prepareDeployCheckoutCredit(stripe, {
        ...base,
        customerId: "cus_clock",
        setupSessionId: "cs_setup",
        store: new MemoryStore(),
      }),
    ).rejects.toThrow("This card setup session is for a different customer.");
  });

  it("skips the credit and still subscribes when the workos user id is invalid", async () => {
    const { stripe, creates } = stripeStub({
      cards: [{ id: "pm_ok", fingerprint: "fpOk", funding: "credit", wallet: null }],
    });

    await expect(
      prepareDeployCheckoutCredit(stripe, {
        ...base,
        workosUserId: "user_local_admin",
        customerId: "cus_1",
        store: new MemoryStore(),
      }),
    ).resolves.toEqual({
      step: "subscribe",
      customerId: "cus_1",
      credit: null,
      announceSignupCredit: false,
    });
    expect(creates).toEqual([]);

    await expect(
      prepareDeployCheckoutCredit(stripe, {
        ...base,
        workosUserId: "user_local_admin",
        store: new MemoryStore(),
      }),
    ).resolves.toEqual({
      step: "subscribe",
      credit: null,
      announceSignupCredit: false,
    });
  });

  it("requires the setup intent user to equal the session user", async () => {
    const { stripe, creates } = stripeStub({
      setupSession: setupSession(
        { id: "pm_new", fingerprint: "fpNew", funding: "credit", wallet: null },
        "cus_new",
        "user_otherpayer",
      ),
    });

    const result = await prepareDeployCheckoutCredit(stripe, {
      ...base,
      setupSessionId: "cs_setup",
      store: new MemoryStore(),
    });

    expect(result).toEqual({
      step: "subscribe",
      customerId: "cus_new",
      credit: null,
      announceSignupCredit: false,
    });
    expect(creates).toEqual([]);
  });

  it("continues checkout when the grant fails for a reason other than a retry", async () => {
    const errorLog = vi.spyOn(console, "error").mockImplementation(() => undefined);
    const { stripe, creates } = stripeStub({
      createError: new Error("stripe unavailable"),
      setupSession: setupSession({
        id: "pm_new",
        fingerprint: "fpNew",
        funding: "credit",
        wallet: null,
      }),
    });

    const result = await prepareDeployCheckoutCredit(stripe, {
      ...base,
      setupSessionId: "cs_setup",
      store: new MemoryStore(),
    });

    expect(result).toEqual({
      step: "subscribe",
      customerId: "cus_new",
      credit: null,
      announceSignupCredit: false,
    });
    expect(creates).toEqual([]);
    expect(errorLog).toHaveBeenCalledWith(
      "Compute signup credit grant failed; continuing checkout",
      expect.objectContaining({ workspaceId: "ws_1", customerId: "cus_new" }),
    );
    errorLog.mockRestore();
  });

  it("keeps an in-progress claim on the refresh path", async () => {
    const { stripe } = stripeStub({
      createError: new ComputeSignupCreditRetryError("signup credit claim is in progress"),
      cards: [{ id: "pm_ok", fingerprint: "fpOk", funding: "credit", wallet: null }],
    });

    await expect(
      prepareDeployCheckoutCredit(stripe, {
        ...base,
        customerId: "cus_1",
        store: new MemoryStore(),
      }),
    ).rejects.toBeInstanceOf(ComputeSignupCreditRetryError);
  });

  it("grades an eligible card past the first page of saved cards", async () => {
    const cards: Card[] = Array.from({ length: 10 }, (_, index) => ({
      id: `pmblank${index}`,
      fingerprint: null,
      funding: "credit",
      wallet: null,
    }));
    cards.push({ id: "pmok", fingerprint: "fpOk", funding: "credit", wallet: null });
    const { stripe, creates } = stripeStub({ cards });

    const result = await prepareDeployCheckoutCredit(stripe, {
      ...base,
      customerId: "cus_1",
      store: new MemoryStore(),
    });

    expect(result).toMatchObject({
      step: "subscribe",
      announceSignupCredit: true,
      credit: { granted: true, amountCents: 500 },
    });
    expect(creates[0]?.params).toMatchObject({
      metadata: expect.objectContaining({ card_fingerprint: "fpOk" }),
    });
  });
});
