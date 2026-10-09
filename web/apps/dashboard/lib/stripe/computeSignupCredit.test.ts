import Stripe from "stripe";
import { afterEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  billingFindFirst: vi.fn(),
  workspaceFindFirst: vi.fn(),
  stripeEnv: vi.fn(),
}));

vi.mock("@/lib/db", () => ({
  db: {
    query: {
      workspaceBilling: { findFirst: mocks.billingFindFirst },
      workspaces: { findFirst: mocks.workspaceFindFirst },
    },
  },
}));

vi.mock("@/lib/env", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/env")>();
  return { ...actual, stripeEnv: mocks.stripeEnv };
});

import {
  COMPUTE_SIGNUP_CREDIT_CENTS,
  type ComputeSignupCreditInput,
  ComputeSignupCreditRetryError,
  computeSignupCreditIdempotencyKey,
  grantComputeSignupCredit,
  handlePaymentMethodAttached,
  resolveSignupWorkosUserId,
  signupCardEligibility,
} from "./computeSignupCredit";
import type { ComputeSignupCreditClaim } from "./computeSignupCreditClaims";

const NOW_SECONDS = 1_700_000_000;
const NOW_MS = NOW_SECONDS * 1000;
const HOUR_MS = 60 * 60 * 1000;

type StoredTransaction = {
  id: string;
  customer: string;
  amount: number;
  currency: string;
  description?: string;
  metadata?: Record<string, string>;
};

type StoredCustomer = {
  id: string;
  deleted?: boolean;
  created: number;
  currency?: string | null;
  metadata: Record<string, string>;
};

type StoredSession = {
  id?: string;
  mode?: string;
  status?: string | null;
  created?: number;
  expires_at?: number | null;
  client_reference_id?: string | null;
  metadata?: Record<string, string> | null;
  setup_intent?: string | null;
  payment_intent?: string | null;
  subscription?: string | null;
};

function page<T>(items: T[], pageSize = 10): AsyncIterable<T> & { data: T[]; has_more: boolean } {
  const result = {
    data: items.slice(0, pageSize),
    has_more: items.length > pageSize,
    async *[Symbol.asyncIterator]() {
      for (const item of items) {
        yield item;
      }
    },
  };
  return result;
}

type StoredPaymentMethod = {
  id: string;
  object: "payment_method";
  customer: string | null;
  created: number;
  type: string;
  card: {
    fingerprint: string | null;
    funding: string;
    wallet: { type: string } | null;
  } | null;
};

class MemoryClaimStore {
  rows: ComputeSignupCreditClaim[] = [];
  failStamp = false;
  beforeClaim: (() => Promise<void> | void) | null = null;
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
      pk: this.nextPk++,
      stripeBalanceTransactionId: null,
    });
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
    if (
      !row ||
      row.attemptId !== attempt.seenAttemptId ||
      row.stripeBalanceTransactionId !== null
    ) {
      return false;
    }
    if (this.beforeClaim) {
      const hook = this.beforeClaim;
      this.beforeClaim = null;
      await hook();
    }
    if (row.attemptId !== attempt.seenAttemptId || row.stripeBalanceTransactionId !== null) {
      return false;
    }
    const workspaceTaken = this.rows.some(
      (existing) => existing.pk !== row.pk && existing.workspaceId === attempt.workspaceId,
    );
    const fingerprintTaken = this.rows.some(
      (existing) => existing.pk !== row.pk && existing.cardFingerprint === attempt.cardFingerprint,
    );
    const userTaken = this.rows.some(
      (existing) => existing.pk !== row.pk && existing.workosUserId === attempt.workosUserId,
    );
    if (workspaceTaken || fingerprintTaken || userTaken) {
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
    if (this.failStamp) {
      return 0;
    }
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

  async deleteAttempt(claim: {
    attemptId: string;
    workspaceId: string;
    cardFingerprint: string;
    workosUserId: string;
  }) {
    this.rows = this.rows.filter(
      (row) =>
        !(
          row.attemptId === claim.attemptId &&
          row.workspaceId === claim.workspaceId &&
          row.cardFingerprint === claim.cardFingerprint &&
          row.workosUserId === claim.workosUserId &&
          row.stripeBalanceTransactionId === null
        ),
    );
  }
}

class FakeStripe {
  transactions: StoredTransaction[] = [];
  creates: Array<Record<string, unknown>> = [];
  idempotencyKeys: Array<string | undefined> = [];
  customers = new Map<string, StoredCustomer>();
  paymentMethods = new Map<string, StoredPaymentMethod>();
  sessions: StoredSession[] = [];
  setupIntents: Array<{
    id: string;
    metadata?: Record<string, string> | null;
    payment_method?: string | null;
  }> = [];
  paymentIntents = new Map<string, { payment_method?: string | null; customer?: string | null }>();
  missingPaymentIntents = new Set<string>();
  subscriptionLists: Array<Record<string, unknown> | undefined> = [];
  subscriptionRetrieves: Array<Record<string, unknown> | undefined> = [];
  subscriptions = new Map<
    string,
    {
      metadata?: Record<string, string> | null;
      default_payment_method?: string | null;
      latest_invoice?: {
        default_payment_method?: string | null;
        payments?: {
          data: Array<{
            payment?: {
              payment_intent?: string | { payment_method?: string | null } | null;
            } | null;
          }>;
        } | null;
      } | null;
    }
  >();
  onListCustomer: ((customerId: string) => void | Promise<void>) | null = null;
  onCreate: (() => void | Promise<void>) | null = null;
  private nextTxn = 1;

  addCustomer(
    id: string,
    metadata: Record<string, string> = {},
    created = NOW_SECONDS - 10,
    currency: string | null = null,
  ) {
    this.customers.set(id, { id, created, metadata, currency });
  }

  client(): Stripe {
    return {
      customers: {
        createBalanceTransaction: (
          customerId: string,
          params: {
            amount: number;
            currency: string;
            description?: string;
            metadata?: Record<string, string>;
          },
          options?: { idempotencyKey?: string },
        ) => this.createBalanceTransaction(customerId, params, options?.idempotencyKey),
        listBalanceTransactions: (customerId: string) => this.listBalanceTransactions(customerId),
        retrieve: (id: string) => this.retrieveCustomer(id),
      },
      paymentMethods: {
        retrieve: async (id: string) => {
          const paymentMethod = this.paymentMethods.get(id);
          if (!paymentMethod) {
            throw new Error(`missing payment method ${id}`);
          }
          return paymentMethod;
        },
      },
      checkout: {
        sessions: {
          list: () => page(this.sessions),
        },
      },
      setupIntents: {
        list: () => page(this.setupIntents),
        retrieve: async (id: string) => {
          const intent = this.setupIntents.find((item) => item.id === id);
          if (!intent) {
            throw new Error(`missing setup intent ${id}`);
          }
          return intent;
        },
      },
      paymentIntents: {
        retrieve: async (id: string) => {
          if (this.missingPaymentIntents.has(id)) {
            throw new Stripe.errors.StripeInvalidRequestError({
              message: `No such payment_intent: '${id}'`,
              type: "invalid_request_error",
              code: "resource_missing",
              statusCode: 404,
            });
          }
          const intent = this.paymentIntents.get(id);
          return {
            id,
            metadata: null,
            payment_method: intent?.payment_method ?? null,
            customer: intent?.customer ?? null,
          };
        },
      },
      subscriptions: {
        list: (params?: Record<string, unknown>) => {
          this.subscriptionLists.push(params);
          return page(
            [...this.subscriptions.entries()].map(([id, subscription]) => ({
              id,
              ...subscription,
            })),
          );
        },
        retrieve: async (id: string, params?: Record<string, unknown>) => {
          this.subscriptionRetrieves.push(params);
          const subscription = this.subscriptions.get(id);
          if (!subscription) {
            throw new Error(`missing subscription ${id}`);
          }
          return { id, ...subscription };
        },
      },
    } as unknown as Stripe;
  }

  private async createBalanceTransaction(
    customerId: string,
    params: {
      amount: number;
      currency: string;
      description?: string;
      metadata?: Record<string, string>;
    },
    idempotencyKey: string | undefined,
  ) {
    this.idempotencyKeys.push(idempotencyKey);
    if (this.onCreate) {
      const hook = this.onCreate;
      this.onCreate = null;
      await hook();
    }
    const transaction: StoredTransaction = {
      id: `cbtxn_${this.nextTxn++}`,
      customer: customerId,
      amount: params.amount,
      currency: params.currency,
      description: params.description,
      metadata: params.metadata,
    };
    this.transactions.push(transaction);
    this.creates.push(params);
    return transaction;
  }

  private listBalanceTransactions(customerId: string) {
    const transactions = this.transactions;
    const hook = this.onListCustomer;
    return (async function* () {
      await hook?.(customerId);
      for (const transaction of transactions.filter(
        (transaction) => transaction.customer === customerId,
      )) {
        yield transaction;
      }
    })();
  }

  private async retrieveCustomer(id: string) {
    const customer = this.customers.get(id);
    if (!customer) {
      throw new Error(`missing customer ${id}`);
    }
    return customer;
  }
}

function input(overrides: Partial<ComputeSignupCreditInput> = {}): ComputeSignupCreditInput {
  return {
    workspaceId: "ws_123",
    customerId: "cus_test",
    fingerprint: "fpABC123",
    workosUserId: "user_123",
    nowMs: NOW_MS,
    ...overrides,
  };
}

function ready() {
  const fake = new FakeStripe();
  fake.addCustomer("cus_test");
  return { fake, store: new MemoryClaimStore(), stripe: fake.client() };
}

describe("signupCardEligibility", () => {
  it("accepts a card fingerprint", () => {
    expect(signupCardEligibility("fpABC123")).toEqual({
      eligible: true,
      fingerprint: "fpABC123",
    });
  });

  it("refuses a missing or invalid fingerprint", () => {
    expect(signupCardEligibility(null)).toEqual({
      eligible: false,
      reason: "card has no fingerprint",
    });
    expect(signupCardEligibility("fp_ok")).toEqual({
      eligible: false,
      reason: "card has no fingerprint",
    });
  });
});

describe("grantComputeSignupCredit", () => {
  it("credits the customer balance by $5 with a workspace idempotency key", async () => {
    const { fake, store, stripe } = ready();

    const result = await grantComputeSignupCredit(stripe, input(), store);

    expect(result).toEqual({
      granted: true,
      transactionId: "cbtxn_1",
      amountCents: COMPUTE_SIGNUP_CREDIT_CENTS,
    });
    expect(fake.creates).toEqual([
      {
        amount: -COMPUTE_SIGNUP_CREDIT_CENTS,
        currency: "usd",
        description: "Compute signup credit",
        metadata: {
          unkey_credit: "compute_signup",
          workspace_id: "ws_123",
          card_fingerprint: "fpABC123",
          workos_user_id: "user_123",
        },
      },
    ]);
    expect(fake.idempotencyKeys).toEqual([computeSignupCreditIdempotencyKey("ws_123")]);
    expect(store.rows[0]).toMatchObject({
      workspaceId: "ws_123",
      cardFingerprint: "fpABC123",
      workosUserId: "user_123",
      stripeBalanceTransactionId: "cbtxn_1",
    });
  });

  it("does not credit again when the same webhook is delivered twice", async () => {
    const { fake, store, stripe } = ready();

    const first = await grantComputeSignupCredit(stripe, input(), store);
    const second = await grantComputeSignupCredit(stripe, input(), store);

    expect(first).toMatchObject({ granted: true, transactionId: "cbtxn_1" });
    expect(second).toEqual({
      granted: false,
      reason: "already credited (cbtxn_1)",
    });
    expect(fake.creates).toHaveLength(1);
    expect(store.rows).toHaveLength(1);
  });

  it("refuses a card fingerprint that already credited another workspace", async () => {
    const { fake, store, stripe } = ready();
    fake.addCustomer("cus_b");
    await grantComputeSignupCredit(stripe, input(), store);

    const result = await grantComputeSignupCredit(
      stripe,
      input({
        workspaceId: "ws_b",
        customerId: "cus_b",
        workosUserId: "user_999",
      }),
      store,
    );

    expect(result).toEqual({
      granted: false,
      reason: "card fingerprint already used",
    });
    expect(fake.creates).toHaveLength(1);
  });

  it("does not credit a second workspace of the same user with a new card", async () => {
    const { fake, store, stripe } = ready();
    fake.addCustomer("cus_b");
    await grantComputeSignupCredit(stripe, input(), store);

    const result = await grantComputeSignupCredit(
      stripe,
      input({
        workspaceId: "ws_b",
        customerId: "cus_b",
        fingerprint: "fpOTHER999",
      }),
      store,
    );

    expect(result).toEqual({
      granted: false,
      reason: "workos user already credited",
    });
    expect(fake.creates).toHaveLength(1);
    expect(store.rows).toHaveLength(1);
  });

  it("does not credit a second workspace of the same user with the same card", async () => {
    const { fake, store, stripe } = ready();
    fake.addCustomer("cus_b");
    await grantComputeSignupCredit(stripe, input(), store);

    const result = await grantComputeSignupCredit(
      stripe,
      input({ workspaceId: "ws_b", customerId: "cus_b" }),
      store,
    );

    expect(result).toEqual({
      granted: false,
      reason: "workos user already credited",
    });
    expect(fake.creates).toHaveLength(1);
    expect(store.rows).toHaveLength(1);
  });

  it("stamps a hidden credit on the same tuple before creating another", async () => {
    const { fake, store, stripe } = ready();
    fake.addCustomer("cus_old");
    store.rows.push({
      pk: 1,
      cardFingerprint: "fpABC123",
      workspaceId: "ws_123",
      stripeCustomerId: "cus_old",
      workosUserId: "user_123",
      stripeBalanceTransactionId: null,
      attemptId: "attempt-old",
      createdAt: NOW_MS - 2 * HOUR_MS,
    });
    fake.transactions.push({
      id: "cbtxn_hidden",
      customer: "cus_old",
      amount: -500,
      currency: "usd",
      metadata: {
        unkey_credit: "compute_signup",
        card_fingerprint: "fpABC123",
        workspace_id: "ws_123",
        workos_user_id: "user_123",
      },
    });

    const result = await grantComputeSignupCredit(stripe, input(), store);

    expect(result).toEqual({
      granted: false,
      reason: "already credited (cbtxn_hidden)",
    });
    expect(fake.creates).toHaveLength(0);
    expect(store.rows).toEqual([
      expect.objectContaining({
        workspaceId: "ws_123",
        cardFingerprint: "fpABC123",
        workosUserId: "user_123",
        stripeCustomerId: "cus_old",
        stripeBalanceTransactionId: "cbtxn_hidden",
      }),
    ]);
  });

  it("keeps the claim when the idempotency key was used with different parameters", async () => {
    const { fake, store, stripe } = ready();
    fake.onCreate = () => {
      throw new Stripe.errors.StripeIdempotencyError({
        message:
          "Keys for idempotent requests can only be used with the same parameters they were first used with.",
        type: "idempotency_error",
        statusCode: 400,
      });
    };

    await expect(grantComputeSignupCredit(stripe, input(), store)).rejects.toBeInstanceOf(
      ComputeSignupCreditRetryError,
    );
    expect(store.rows).toHaveLength(1);
    expect(store.rows[0]?.stripeBalanceTransactionId).toBeNull();

    fake.addCustomer("cus_b");
    await expect(
      grantComputeSignupCredit(stripe, input({ customerId: "cus_b" }), store),
    ).rejects.toBeInstanceOf(ComputeSignupCreditRetryError);
    expect(fake.transactions).toHaveLength(0);
    expect(store.rows).toHaveLength(1);
    expect(store.rows[0]?.stripeBalanceTransactionId).toBeNull();
  });

  it("keeps the claim when the balance transaction fails ambiguously", async () => {
    const { fake, store, stripe } = ready();
    fake.onCreate = () => {
      throw new Stripe.errors.StripeAPIError({
        message: "upstream",
        statusCode: 500,
        type: "api_error",
      });
    };

    await expect(grantComputeSignupCredit(stripe, input(), store)).rejects.toBeInstanceOf(
      Stripe.errors.StripeAPIError,
    );
    expect(store.rows).toHaveLength(1);
    expect(store.rows[0]?.stripeBalanceTransactionId).toBeNull();
  });

  it("deletes the claim only when the create is an invalid request", async () => {
    const { store, stripe, fake } = ready();
    fake.onCreate = () => {
      throw new Stripe.errors.StripeInvalidRequestError({
        message: "bad customer",
        type: "invalid_request_error",
        statusCode: 400,
      });
    };

    await expect(grantComputeSignupCredit(stripe, input(), store)).rejects.toBeInstanceOf(
      Stripe.errors.StripeInvalidRequestError,
    );
    expect(store.rows).toHaveLength(0);
  });

  it("throws when recording the transaction changes no claim row", async () => {
    const { fake, store, stripe } = ready();
    store.failStamp = true;
    const errorLog = vi.spyOn(console, "error").mockImplementation(() => undefined);

    await expect(grantComputeSignupCredit(stripe, input(), store)).rejects.toBeInstanceOf(
      ComputeSignupCreditRetryError,
    );
    expect(fake.transactions).toHaveLength(1);
    expect(store.rows[0]?.stripeBalanceTransactionId).toBeNull();
    expect(errorLog).toHaveBeenCalledWith(
      "Compute signup credit claim conflict while recording the balance transaction",
      expect.objectContaining({
        transactionId: "cbtxn_1",
        workspaceId: "ws_123",
      }),
    );
    errorLog.mockRestore();
  });

  it("records a claim row when a listed signup credit has none", async () => {
    const { fake, store, stripe } = ready();
    fake.transactions.push({
      id: "cbtxn_existing",
      customer: "cus_test",
      amount: -500,
      currency: "usd",
      metadata: {
        unkey_credit: "compute_signup",
        card_fingerprint: "fpABC123",
        workspace_id: "ws_123",
        workos_user_id: "user_123",
      },
    });

    const result = await grantComputeSignupCredit(stripe, input(), store);

    expect(result).toEqual({
      granted: false,
      reason: "already credited (cbtxn_existing)",
    });
    expect(fake.creates).toHaveLength(0);
    expect(store.rows).toEqual([
      expect.objectContaining({
        workspaceId: "ws_123",
        cardFingerprint: "fpABC123",
        workosUserId: "user_123",
        stripeBalanceTransactionId: "cbtxn_existing",
      }),
    ]);
  });

  it("stamps the exact credit when a newer signup credit is for a different card", async () => {
    const { fake, store, stripe } = ready();
    fake.transactions.push(
      {
        id: "cbtxn_partial",
        customer: "cus_test",
        amount: -500,
        currency: "usd",
        metadata: { unkey_credit: "compute_signup" },
      },
      {
        id: "cbtxn_newer",
        customer: "cus_test",
        amount: -500,
        currency: "usd",
        metadata: {
          unkey_credit: "compute_signup",
          card_fingerprint: "fpOTHER999",
          workos_user_id: "user_123",
          workspace_id: "ws_other",
        },
      },
      {
        id: "cbtxn_exact",
        customer: "cus_test",
        amount: -500,
        currency: "usd",
        metadata: {
          unkey_credit: "compute_signup",
          card_fingerprint: "fpABC123",
          workos_user_id: "user_123",
          workspace_id: "ws_123",
        },
      },
    );
    store.rows.push({
      pk: 1,
      cardFingerprint: "fpABC123",
      workspaceId: "ws_123",
      stripeCustomerId: "cus_test",
      workosUserId: "user_123",
      stripeBalanceTransactionId: null,
      attemptId: "attempt-exact",
      createdAt: NOW_MS - 2 * HOUR_MS,
    });

    const result = await grantComputeSignupCredit(stripe, input(), store);

    expect(result).toEqual({
      granted: false,
      reason: "already credited (cbtxn_exact)",
    });
    expect(fake.creates).toHaveLength(0);
    expect(store.rows).toEqual([
      expect.objectContaining({
        cardFingerprint: "fpABC123",
        workosUserId: "user_123",
        stripeBalanceTransactionId: "cbtxn_exact",
      }),
    ]);
  });

  it("does not resume a fresh attempt, then credits once the attempt is stale", async () => {
    const { fake, store, stripe } = ready();
    fake.onCreate = () => {
      throw new Error("crash before stripe accepts the balance transaction");
    };

    await expect(grantComputeSignupCredit(stripe, input(), store)).rejects.toThrow(
      "crash before stripe accepts the balance transaction",
    );
    await expect(grantComputeSignupCredit(stripe, input(), store)).rejects.toBeInstanceOf(
      ComputeSignupCreditRetryError,
    );
    expect(fake.creates).toHaveLength(0);

    const retry = await grantComputeSignupCredit(
      stripe,
      input({ nowMs: NOW_MS + 2 * HOUR_MS }),
      store,
    );
    expect(retry).toMatchObject({ granted: true, transactionId: "cbtxn_1" });
    expect(fake.creates).toHaveLength(1);
  });

  it("leaves a foreign unfinished row in place and returns without retrying", async () => {
    const { fake, store, stripe } = ready();
    const errorLog = vi.spyOn(console, "error").mockImplementation(() => undefined);
    const rows = [
      {
        pk: 1,
        cardFingerprint: "fpOLD999",
        workspaceId: "ws_123",
        stripeCustomerId: "cus_test",
        workosUserId: "user_123",
        stripeBalanceTransactionId: null,
        attemptId: "attempt-ours",
        createdAt: NOW_MS - 2 * HOUR_MS,
      },
      {
        pk: 2,
        cardFingerprint: "fpABC123",
        workspaceId: "ws_other",
        stripeCustomerId: "cus_other",
        workosUserId: "user_other1",
        stripeBalanceTransactionId: null,
        attemptId: "attempt-theirs",
        createdAt: NOW_MS - 2 * HOUR_MS,
      },
    ];
    store.rows.push(...rows);

    const result = await grantComputeSignupCredit(stripe, input(), store);

    expect(result).toEqual({
      granted: false,
      reason: "signup credit claim conflicts with another row",
    });
    expect(fake.transactions).toHaveLength(0);
    expect(store.rows).toEqual(rows);
    expect(errorLog).toHaveBeenCalledWith(
      "Compute signup credit left unfinished for manual reconcile",
      expect.objectContaining({ workspaceId: "ws_123", cardFingerprint: "fpABC123" }),
    );
    errorLog.mockRestore();
  });

  it("does not retake a different workspace, card, or user even after the row is old", async () => {
    const { fake, store, stripe } = ready();
    fake.addCustomer("cus_taker");
    const errorLog = vi.spyOn(console, "error").mockImplementation(() => undefined);
    const row = {
      pk: 1,
      cardFingerprint: "fpABC123",
      workspaceId: "ws_owner",
      stripeCustomerId: "cus_owner",
      workosUserId: "user_owner",
      stripeBalanceTransactionId: null,
      attemptId: "attempt-stale",
      createdAt: NOW_MS - 72 * HOUR_MS,
    };
    store.rows.push(row);

    const result = await grantComputeSignupCredit(
      stripe,
      input({
        workspaceId: "ws_taker",
        customerId: "cus_taker",
        workosUserId: "user_taker",
        nowMs: NOW_MS + 72 * HOUR_MS,
      }),
      store,
    );

    expect(result).toEqual({
      granted: false,
      reason: "signup credit claim conflicts with another row",
    });
    expect(fake.transactions).toHaveLength(0);
    expect(store.rows).toEqual([row]);
    expect(errorLog).toHaveBeenCalledWith(
      "Compute signup credit left unfinished for manual reconcile",
      expect.objectContaining({ workspaceId: "ws_taker" }),
    );
    errorLog.mockRestore();
  });

  it("updates stripe_customer_id when the same tuple resumes", async () => {
    const { fake, store, stripe } = ready();
    fake.addCustomer("cus_old");
    store.rows.push({
      pk: 1,
      cardFingerprint: "fpABC123",
      workspaceId: "ws_123",
      stripeCustomerId: "cus_old",
      workosUserId: "user_123",
      stripeBalanceTransactionId: null,
      attemptId: "attempt-old",
      createdAt: NOW_MS - 2 * HOUR_MS,
    });

    const result = await grantComputeSignupCredit(stripe, input(), store);

    expect(result).toMatchObject({ granted: true, transactionId: "cbtxn_1" });
    expect(store.rows[0]).toMatchObject({
      workspaceId: "ws_123",
      cardFingerprint: "fpABC123",
      workosUserId: "user_123",
      stripeCustomerId: "cus_test",
      stripeBalanceTransactionId: "cbtxn_1",
    });
    expect(store.rows[0]?.attemptId).not.toBe("attempt-old");
  });

  it("does not compare-and-set a same-tuple attempt that changed after it was judged", async () => {
    const { fake, store, stripe } = ready();
    store.rows.push({
      pk: 1,
      cardFingerprint: "fpABC123",
      workspaceId: "ws_123",
      stripeCustomerId: "cus_test",
      workosUserId: "user_123",
      stripeBalanceTransactionId: null,
      attemptId: "attempt-stale",
      createdAt: NOW_MS - 2 * HOUR_MS,
    });
    fake.onListCustomer = () => {
      const row = store.rows[0];
      if (!row) {
        return;
      }
      row.attemptId = "attempt-fresh";
      row.createdAt = NOW_MS;
    };

    await expect(grantComputeSignupCredit(stripe, input(), store)).rejects.toBeInstanceOf(
      ComputeSignupCreditRetryError,
    );
    expect(fake.transactions).toHaveLength(0);
    expect(store.rows[0]).toMatchObject({
      workspaceId: "ws_123",
      workosUserId: "user_123",
      cardFingerprint: "fpABC123",
      stripeCustomerId: "cus_test",
      attemptId: "attempt-fresh",
      stripeBalanceTransactionId: null,
    });
  });

  it("lets only one of two takers win when both read the attempt before either compare-and-set", async () => {
    const { fake, store, stripe } = ready();
    store.rows.push({
      pk: 1,
      cardFingerprint: "fpABC123",
      workspaceId: "ws_123",
      stripeCustomerId: "cus_test",
      workosUserId: "user_123",
      stripeBalanceTransactionId: null,
      attemptId: "attempt-seen",
      createdAt: NOW_MS - 2 * HOUR_MS,
    });
    store.beforeClaim = async () => {
      const other = await grantComputeSignupCredit(stripe, input(), store);
      expect(other).toMatchObject({ granted: true });
    };

    await expect(grantComputeSignupCredit(stripe, input(), store)).rejects.toBeInstanceOf(
      ComputeSignupCreditRetryError,
    );
    expect(fake.transactions).toHaveLength(1);
  });

  it("grants once when two workspaces of one user attach at the same time", async () => {
    const { fake, store, stripe } = ready();
    fake.addCustomer("cus_b");
    fake.addCustomer("cus_c");
    store.beforeClaim = async () => {
      await expect(
        grantComputeSignupCredit(
          stripe,
          input({
            workspaceId: "ws_c",
            customerId: "cus_c",
            fingerprint: "fpOTHER999",
          }),
          store,
        ),
      ).resolves.toEqual({
        granted: false,
        reason: "signup credit claim conflicts with another row",
      });
    };

    const result = await grantComputeSignupCredit(
      stripe,
      input({
        workspaceId: "ws_b",
        customerId: "cus_b",
        fingerprint: "fpNEW999",
      }),
      store,
    );

    expect(result).toMatchObject({ granted: true });
    expect(fake.transactions).toHaveLength(1);
    expect(store.rows).toHaveLength(1);
    expect(store.rows[0]).toMatchObject({ workosUserId: "user_123" });
  });

  it("retries when the stored transaction's attempt id is not the attempt that was held", async () => {
    const { fake, store, stripe } = ready();
    fake.onCreate = () => {
      const row = store.rows[0];
      if (row) {
        row.attemptId = "attempt-winner";
        row.stripeBalanceTransactionId = "cbtxn_1";
      }
    };

    await expect(grantComputeSignupCredit(stripe, input(), store)).rejects.toBeInstanceOf(
      ComputeSignupCreditRetryError,
    );
    expect(fake.transactions).toHaveLength(1);
    expect(store.rows[0]?.attemptId).toBe("attempt-winner");
  });

  it("requests a usd balance transaction when the customer currency is eur", async () => {
    const { fake, store, stripe } = ready();
    fake.addCustomer("cus_eur", {}, NOW_SECONDS - 10, "eur");

    const result = await grantComputeSignupCredit(stripe, input({ customerId: "cus_eur" }), store);

    expect(result).toMatchObject({ granted: true, amountCents: 500 });
    expect(fake.creates).toEqual([expect.objectContaining({ amount: -500, currency: "usd" })]);
    expect(store.rows).toHaveLength(1);
  });

  it("deletes the held claim when the balance transaction rejects the currency", async () => {
    const { fake, store, stripe } = ready();
    fake.onCreate = () => {
      throw new Stripe.errors.StripeInvalidRequestError({
        message: "Balance transaction currency must match the customer currency",
        type: "invalid_request_error",
        statusCode: 400,
      });
    };

    await expect(grantComputeSignupCredit(stripe, input(), store)).rejects.toBeInstanceOf(
      Stripe.errors.StripeInvalidRequestError,
    );
    expect(store.rows).toHaveLength(0);

    fake.addCustomer("cus_b");
    const other = await grantComputeSignupCredit(
      stripe,
      input({
        workspaceId: "ws_b",
        customerId: "cus_b",
        fingerprint: "fpOTHER999",
        nowMs: NOW_MS + 72 * HOUR_MS,
      }),
      store,
    );

    expect(other).toMatchObject({ granted: true });
    expect(store.rows).toHaveLength(1);
    expect(store.rows[0]).toMatchObject({
      workspaceId: "ws_b",
      workosUserId: "user_123",
      cardFingerprint: "fpOTHER999",
    });
  });

  it("does not drop an in-flight claim when another delivery arrives before the credit lands", async () => {
    const { fake, store, stripe } = ready();
    fake.addCustomer("cus_eur", {}, NOW_SECONDS - 10, "eur");
    fake.addCustomer("cus_b");
    fake.onCreate = async () => {
      const attemptId = store.rows[0]?.attemptId;
      await expect(
        grantComputeSignupCredit(stripe, input({ customerId: "cus_eur" }), store),
      ).rejects.toBeInstanceOf(ComputeSignupCreditRetryError);
      expect(store.rows).toEqual([
        expect.objectContaining({
          attemptId,
          stripeBalanceTransactionId: null,
        }),
      ]);
      expect(fake.transactions).toHaveLength(0);
    };

    const created = await grantComputeSignupCredit(stripe, input(), store);

    expect(created).toMatchObject({ granted: true, transactionId: "cbtxn_1", amountCents: 500 });
    expect(fake.transactions).toEqual([
      expect.objectContaining({ customer: "cus_test", amount: -500 }),
    ]);
    expect(store.rows).toEqual([
      expect.objectContaining({
        workspaceId: "ws_123",
        cardFingerprint: "fpABC123",
        workosUserId: "user_123",
        stripeBalanceTransactionId: "cbtxn_1",
      }),
    ]);

    const later = await grantComputeSignupCredit(
      stripe,
      input({
        customerId: "cus_b",
        nowMs: NOW_MS + 72 * HOUR_MS,
      }),
      store,
    );

    expect(later).toEqual({ granted: false, reason: "already credited (cbtxn_1)" });
    expect(fake.transactions).toHaveLength(1);
    expect(fake.creates).toHaveLength(1);
  });

  it("leaves a stranded claim for the same tuple to resume", async () => {
    const { fake, store, stripe } = ready();
    fake.addCustomer("cus_eur", {}, NOW_SECONDS - 10, "eur");
    store.rows.push({
      pk: 1,
      cardFingerprint: "fpABC123",
      workspaceId: "ws_123",
      stripeCustomerId: "cus_eur",
      workosUserId: "user_123",
      stripeBalanceTransactionId: null,
      attemptId: "attempt-currency",
      createdAt: NOW_MS,
    });

    await expect(
      grantComputeSignupCredit(stripe, input({ customerId: "cus_eur" }), store),
    ).rejects.toBeInstanceOf(ComputeSignupCreditRetryError);
    expect(fake.creates).toHaveLength(0);
    expect(store.rows).toEqual([
      expect.objectContaining({
        attemptId: "attempt-currency",
        stripeBalanceTransactionId: null,
      }),
    ]);

    fake.addCustomer("cus_usd");
    const resumed = await grantComputeSignupCredit(
      stripe,
      input({
        customerId: "cus_usd",
        nowMs: NOW_MS + 2 * HOUR_MS,
      }),
      store,
    );

    expect(resumed).toMatchObject({ granted: true, transactionId: "cbtxn_1" });
    expect(fake.transactions).toHaveLength(1);
    expect(store.rows).toHaveLength(1);
  });

  it("stamps a landed credit on the stored customer when a later attach is not usd", async () => {
    const { fake, store, stripe } = ready();
    fake.addCustomer("cus_usd");
    fake.addCustomer("cus_eur", {}, NOW_SECONDS - 10, "eur");
    fake.addCustomer("cus_usd2");
    fake.transactions.push({
      id: "cbtxn_landed",
      customer: "cus_usd",
      amount: -500,
      currency: "usd",
      metadata: {
        unkey_credit: "compute_signup",
        card_fingerprint: "fpABC123",
        workspace_id: "ws_123",
        workos_user_id: "user_123",
      },
    });
    store.rows.push({
      pk: 1,
      cardFingerprint: "fpABC123",
      workspaceId: "ws_123",
      stripeCustomerId: "cus_usd",
      workosUserId: "user_123",
      stripeBalanceTransactionId: null,
      attemptId: "attempt-stranded",
      createdAt: NOW_MS,
    });

    await expect(
      grantComputeSignupCredit(stripe, input({ customerId: "cus_eur" }), store),
    ).rejects.toBeInstanceOf(ComputeSignupCreditRetryError);
    expect(fake.creates).toHaveLength(0);
    expect(store.rows).toEqual([
      expect.objectContaining({
        attemptId: "attempt-stranded",
        stripeCustomerId: "cus_usd",
        stripeBalanceTransactionId: null,
      }),
    ]);

    const resumed = await grantComputeSignupCredit(
      stripe,
      input({
        customerId: "cus_usd",
        nowMs: NOW_MS + 2 * HOUR_MS,
      }),
      store,
    );

    expect(resumed).toEqual({ granted: false, reason: "already credited (cbtxn_landed)" });
    expect(fake.creates).toHaveLength(0);
    expect(store.rows).toEqual([
      expect.objectContaining({
        stripeCustomerId: "cus_usd",
        stripeBalanceTransactionId: "cbtxn_landed",
      }),
    ]);

    const later = await grantComputeSignupCredit(
      stripe,
      input({
        workspaceId: "ws_b",
        customerId: "cus_usd2",
        fingerprint: "fpOTHER999",
        nowMs: NOW_MS + 72 * HOUR_MS,
      }),
      store,
    );

    expect(later).toEqual({ granted: false, reason: "workos user already credited" });
    expect(fake.creates).toHaveLength(0);
    expect(fake.transactions).toHaveLength(1);
  });

  it("keeps a recorded credit when the customer currency is not usd", async () => {
    const { fake, store, stripe } = ready();
    fake.addCustomer("cus_eur", {}, NOW_SECONDS - 10, "eur");
    store.rows.push({
      pk: 1,
      cardFingerprint: "fpABC123",
      workspaceId: "ws_123",
      stripeCustomerId: "cus_eur",
      workosUserId: "user_123",
      stripeBalanceTransactionId: "cbtxn_done",
      attemptId: "attempt-done",
      createdAt: NOW_MS,
    });

    const result = await grantComputeSignupCredit(stripe, input({ customerId: "cus_eur" }), store);

    expect(result).toEqual({ granted: false, reason: "already credited (cbtxn_done)" });
    expect(store.rows).toEqual([
      expect.objectContaining({
        attemptId: "attempt-done",
        stripeBalanceTransactionId: "cbtxn_done",
      }),
    ]);
  });
});

describe("handlePaymentMethodAttached", () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  function attached(card: {
    fingerprint: string | null;
    funding: string;
    wallet?: { type: string } | null;
  }) {
    const fake = new FakeStripe();
    fake.addCustomer("cus_test", {}, NOW_SECONDS - 10);
    fake.paymentMethods.set("pm_card", {
      id: "pm_card",
      object: "payment_method",
      customer: "cus_test",
      created: NOW_SECONDS,
      type: "card",
      card: {
        fingerprint: card.fingerprint,
        funding: card.funding,
        wallet: card.wallet ?? null,
      },
    });
    const store = new MemoryClaimStore();
    mocks.stripeEnv.mockReturnValue({ STRIPE_SECRET_KEY: "sk_test" });
    mocks.billingFindFirst.mockResolvedValue({
      workspaceId: "ws_123",
      workspace: { deletedAtM: null },
    });
    return { fake, store, stripe: fake.client() };
  }

  async function handle(
    setup: ReturnType<typeof attached>,
    customer?: string | null,
    resolveUserId: () => Promise<string | null> = async () => "user_123",
  ) {
    return handlePaymentMethodAttached(
      setup.stripe,
      {
        id: "pm_card",
        object: "payment_method",
        ...(customer !== undefined ? { customer } : {}),
      },
      { store: setup.store, nowSeconds: NOW_SECONDS, resolveUserId },
    );
  }

  it("credits a prepaid card and a wallet card", async () => {
    const prepaid = attached({ fingerprint: "fpPrepaid", funding: "prepaid" });
    const wallet = attached({
      fingerprint: "fpWallet",
      funding: "credit",
      wallet: { type: "apple_pay" },
    });

    await expect(handle(prepaid)).resolves.toMatchObject({ granted: true, amountCents: 500 });
    await expect(handle(wallet)).resolves.toMatchObject({ granted: true, amountCents: 500 });
    expect(prepaid.fake.creates[0]).toMatchObject({ amount: -500, currency: "usd" });
    expect(wallet.fake.creates[0]).toMatchObject({ amount: -500, currency: "usd" });
  });

  it("credits a saved debit card", async () => {
    const setup = attached({ fingerprint: "fpABC123", funding: "debit" });

    const result = await handle(setup);

    expect(result).toMatchObject({
      granted: true,
      amountCents: 500,
      transactionId: "cbtxn_1",
    });
    expect(setup.fake.creates[0]).toMatchObject({ amount: -500 });
  });

  it("stamps a listed credit when the saved card was detached before the retry", async () => {
    const setup = attached({ fingerprint: "fpABC123", funding: "credit" });
    const paymentMethod = setup.fake.paymentMethods.get("pm_card");
    if (!paymentMethod) {
      throw new Error("missing payment method");
    }
    paymentMethod.customer = null;
    setup.fake.transactions.push({
      id: "cbtxn_hidden",
      customer: "cus_test",
      amount: -500,
      currency: "usd",
      metadata: {
        unkey_credit: "compute_signup",
        card_fingerprint: "fpABC123",
        workspace_id: "ws_123",
        workos_user_id: "user_123",
      },
    });

    const result = await handle(setup, "cus_test");

    expect(result).toEqual({
      granted: false,
      reason: "already credited (cbtxn_hidden)",
    });
    expect(setup.fake.creates).toHaveLength(0);
    expect(setup.store.rows[0]).toMatchObject({
      workspaceId: "ws_123",
      workosUserId: "user_123",
      stripeBalanceTransactionId: "cbtxn_hidden",
    });
  });

  it("retries while a new customer has no workspace yet", async () => {
    const setup = attached({ fingerprint: "fpABC123", funding: "credit" });
    mocks.billingFindFirst.mockResolvedValue(undefined);
    mocks.workspaceFindFirst.mockResolvedValue(undefined);

    await expect(handle(setup)).rejects.toBeInstanceOf(ComputeSignupCreditRetryError);
    expect(setup.fake.creates).toHaveLength(0);
  });

  it("skips with 200 when no exact user id can be resolved", async () => {
    const setup = attached({ fingerprint: "fpABC123", funding: "credit" });
    const infoLog = vi.spyOn(console, "info").mockImplementation(() => undefined);

    const result = await handle(setup, undefined, async () => null);

    expect(result).toEqual({ granted: false, reason: "signup user not found" });
    expect(setup.fake.creates).toHaveLength(0);
    expect(infoLog).toHaveBeenCalledWith(
      "Compute signup credit skipped: card has no exact workos user",
      expect.objectContaining({
        workspaceId: "ws_123",
        customerId: "cus_test",
        paymentMethodId: "pm_card",
      }),
    );
    infoLog.mockRestore();
  });
});

describe("resolveSignupWorkosUserId", () => {
  afterEach(() => {
    vi.clearAllMocks();
  });

  it("uses the SetupIntent metadata for this card", async () => {
    const { fake, stripe } = ready();
    fake.setupIntents.push(
      {
        id: "seti_other",
        metadata: { workos_user_id: "user_othercard" },
        payment_method: "pm_other",
      },
      {
        id: "seti_1",
        metadata: { workos_user_id: "user_setup" },
        payment_method: "pm_card",
      },
    );

    await expect(resolveSignupWorkosUserId(stripe, "cus_test", "pm_card", "ws_123")).resolves.toBe(
      "user_setup",
    );
  });

  it("uses the subscription user when default_payment_method is this card", async () => {
    const { fake, stripe } = ready();
    const customer = fake.customers.get("cus_test");
    if (!customer) {
      throw new Error("missing customer");
    }
    customer.metadata = { workos_user_id: "user_portaladmin" };
    fake.sessions.push({
      id: "cs_open",
      mode: "subscription",
      status: "open",
      subscription: null,
      metadata: { workos_user_id: "user_loose" },
    });
    fake.subscriptions.set("sub_other", {
      metadata: { workos_user_id: "user_other" },
      default_payment_method: "pm_old",
    });
    fake.subscriptions.set("sub_1", {
      metadata: { workos_user_id: "user_payer" },
      default_payment_method: "pm_card",
    });

    await expect(resolveSignupWorkosUserId(stripe, "cus_test", "pm_card", "ws_123")).resolves.toBe(
      "user_payer",
    );
  });

  it("returns null when no SetupIntent or subscription payment method matches", async () => {
    const { fake, stripe } = ready();
    const customer = fake.customers.get("cus_test");
    if (!customer) {
      throw new Error("missing customer");
    }
    customer.metadata = { workos_user_id: "user_portaladmin" };
    fake.sessions.push({
      id: "cs_loose",
      mode: "subscription",
      status: "complete",
      subscription: "sub_loose",
      metadata: { workos_user_id: "user_loose" },
    });
    fake.subscriptions.set("sub_loose", {
      metadata: { workos_user_id: "user_loose" },
      default_payment_method: null,
    });

    await expect(
      resolveSignupWorkosUserId(stripe, "cus_test", "pm_card", "ws_123"),
    ).resolves.toBeNull();
  });

  it("uses a string payment intent on the latest invoice", async () => {
    const { fake, stripe } = ready();
    fake.paymentIntents.set("pi_1", { payment_method: "pm_card", customer: "cus_test" });
    fake.subscriptions.set("sub_1", {
      metadata: { workos_user_id: "user_payer" },
      default_payment_method: null,
      latest_invoice: {
        payments: {
          data: [{ payment: { payment_intent: "pi_1" } }],
        },
      },
    });

    await expect(resolveSignupWorkosUserId(stripe, "cus_test", "pm_card", "ws_123")).resolves.toBe(
      "user_payer",
    );
    expect(fake.subscriptionLists).toContainEqual(
      expect.objectContaining({
        customer: "cus_test",
        expand: ["data.latest_invoice.payments"],
      }),
    );
  });

  it("ignores a payment intent for a different customer", async () => {
    const { fake, stripe } = ready();
    fake.paymentIntents.set("pi_1", { payment_method: "pm_card", customer: "cus_other" });
    fake.subscriptions.set("sub_1", {
      metadata: { workos_user_id: "user_payer" },
      default_payment_method: null,
      latest_invoice: {
        payments: {
          data: [{ payment: { payment_intent: "pi_1" } }],
        },
      },
    });

    await expect(
      resolveSignupWorkosUserId(stripe, "cus_test", "pm_card", "ws_123"),
    ).resolves.toBeNull();
  });

  it("skips a missing payment intent without retrying", async () => {
    const { fake, stripe } = ready();
    fake.missingPaymentIntents.add("pi_gone");
    fake.subscriptions.set("sub_1", {
      metadata: { workos_user_id: "user_payer" },
      default_payment_method: null,
      latest_invoice: {
        payments: {
          data: [{ payment: { payment_intent: "pi_gone" } }],
        },
      },
    });
    const warnLog = vi.spyOn(console, "warn").mockImplementation(() => undefined);

    await expect(
      resolveSignupWorkosUserId(stripe, "cus_test", "pm_card", "ws_123"),
    ).resolves.toBeNull();
    expect(warnLog).toHaveBeenCalledWith(
      "Compute signup credit skipped: payment intent is missing",
      expect.objectContaining({
        customerId: "cus_test",
        paymentIntentId: "pi_gone",
      }),
    );
    warnLog.mockRestore();
  });

  it("uses the subscription user when the setup intent user id is invalid", async () => {
    const { fake, stripe } = ready();
    fake.setupIntents.push({
      id: "seti_bad",
      metadata: { workos_user_id: "not-a-workos-user" },
      payment_method: "pm_card",
    });
    fake.subscriptions.set("sub_payer", {
      metadata: { workos_user_id: "user_payer" },
      default_payment_method: "pm_card",
    });

    await expect(resolveSignupWorkosUserId(stripe, "cus_test", "pm_card", "ws_123")).resolves.toBe(
      "user_payer",
    );
  });

  it("retries a card first seen during an open subscription checkout", async () => {
    const { fake, stripe } = ready();
    fake.sessions.push({
      id: "cs_open",
      mode: "subscription",
      status: "open",
      created: NOW_SECONDS - 20 * 60 * 60,
      expires_at: NOW_SECONDS + 4 * 60 * 60,
      client_reference_id: "ws_123",
      subscription: null,
    });

    await expect(
      resolveSignupWorkosUserId(stripe, "cus_test", "pm_card", "ws_123", NOW_SECONDS),
    ).rejects.toBeInstanceOf(ComputeSignupCreditRetryError);
  });

  it("does not retry a card created before the checkout session", async () => {
    const { fake, stripe } = ready();
    fake.sessions.push({
      id: "cs_open",
      mode: "subscription",
      status: "open",
      created: NOW_SECONDS + 60,
      expires_at: NOW_SECONDS + 24 * 60 * 60,
      client_reference_id: "ws_123",
      subscription: null,
    });

    await expect(
      resolveSignupWorkosUserId(stripe, "cus_test", "pm_card", "ws_123", NOW_SECONDS),
    ).resolves.toBeNull();
  });

  it("does not retry after the checkout session has expired", async () => {
    const { fake, stripe } = ready();
    fake.sessions.push({
      id: "cs_expired",
      mode: "subscription",
      status: "expired",
      created: NOW_SECONDS - 26 * 60 * 60,
      expires_at: NOW_SECONDS - 2 * 60 * 60,
      client_reference_id: "ws_123",
      subscription: null,
    });

    await expect(
      resolveSignupWorkosUserId(stripe, "cus_test", "pm_card", "ws_123", NOW_SECONDS - 3 * 60 * 60),
    ).resolves.toBeNull();
  });

  it("grants a later subscription card after a card with no fingerprint", async () => {
    const { fake, store, stripe } = ready();
    fake.paymentMethods.set("pm_blank", {
      id: "pm_blank",
      object: "payment_method",
      customer: "cus_test",
      created: NOW_SECONDS,
      type: "card",
      card: { fingerprint: null, funding: "credit", wallet: null },
    });
    fake.setupIntents.push({
      id: "seti_blank",
      metadata: { workos_user_id: "user_123" },
      payment_method: "pm_blank",
    });
    mocks.stripeEnv.mockReturnValue({ STRIPE_SECRET_KEY: "sk_test" });
    mocks.billingFindFirst.mockResolvedValue({
      workspaceId: "ws_123",
      workspace: { deletedAtM: null },
    });

    const refused = await handlePaymentMethodAttached(
      stripe,
      { id: "pm_blank", object: "payment_method" },
      { store, nowSeconds: NOW_SECONDS },
    );
    expect(refused).toEqual({ granted: false, reason: "card has no fingerprint" });
    expect(fake.creates).toHaveLength(0);

    fake.paymentMethods.set("pm_checkout", {
      id: "pm_checkout",
      object: "payment_method",
      customer: "cus_test",
      created: NOW_SECONDS + 30,
      type: "card",
      card: { fingerprint: "fpCheckout", funding: "credit", wallet: null },
    });
    fake.sessions.push({
      id: "cs_sub",
      mode: "subscription",
      status: "open",
      created: NOW_SECONDS,
      expires_at: NOW_SECONDS + 24 * 60 * 60,
      client_reference_id: "ws_123",
      subscription: "sub_open",
    });
    fake.subscriptions.set("sub_open", {
      metadata: { workos_user_id: "user_123" },
      default_payment_method: null,
    });

    await expect(
      handlePaymentMethodAttached(
        stripe,
        { id: "pm_checkout", object: "payment_method" },
        { store, nowSeconds: NOW_SECONDS + 30 },
      ),
    ).rejects.toBeInstanceOf(ComputeSignupCreditRetryError);
    expect(fake.creates).toHaveLength(0);

    const openSession = fake.sessions[0];
    if (!openSession) {
      throw new Error("missing checkout session");
    }
    openSession.status = "complete";
    fake.subscriptions.set("sub_open", {
      metadata: { workos_user_id: "user_123" },
      default_payment_method: "pm_checkout",
    });

    const granted = await handlePaymentMethodAttached(
      stripe,
      { id: "pm_checkout", object: "payment_method" },
      { store, nowSeconds: NOW_SECONDS + 40 },
    );
    expect(granted).toMatchObject({
      granted: true,
      amountCents: 500,
      transactionId: "cbtxn_1",
    });
    expect(fake.creates[0]).toMatchObject({
      amount: -500,
      metadata: expect.objectContaining({
        card_fingerprint: "fpCheckout",
        workos_user_id: "user_123",
      }),
    });
  });
});
