import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  alertCustomerLifecycle,
  alertInvalidProductQuotaMetadata,
  alertOrphanedDeploySubscription,
  alertPaymentFailed,
  alertPaymentRecovered,
  escapeSlackText,
  mrkdwn,
  stripeCustomerUrl,
} from "./slackAlerts";

describe("escapeSlackText", () => {
  it("escapes the characters slack treats as markup", () => {
    expect(escapeSlackText("<https://evil.example/invoice|View your invoice>")).toBe(
      "&lt;https://evil.example/invoice|View your invoice&gt;",
    );
    expect(escapeSlackText("Tom & Jerry")).toBe("Tom &amp; Jerry");
    expect(escapeSlackText("<a> & <b>")).toBe("&lt;a&gt; &amp; &lt;b&gt;");
  });

  it("escapes ampersands first so the entities it emits are not re-escaped", () => {
    // Escaping "<" last would yield "&amp;lt;" here, because the "&" introduced by the
    // angle-bracket rule would still be ahead of the ampersand rule.
    expect(escapeSlackText("<")).toBe("&lt;");
    expect(escapeSlackText(">")).toBe("&gt;");
  });

  it("collapses newlines so a value cannot forge an extra line in the alert", () => {
    expect(escapeSlackText("Acme\n\n:white_check_mark: Payment recovered, no action needed")).toBe(
      "Acme :white_check_mark: Payment recovered, no action needed",
    );
    expect(escapeSlackText("Acme\r\n\tInc")).toBe("Acme Inc");
  });

  /**
   * A plain \n is not the only way to forge a line: some renderers break on the Unicode line
   * and paragraph separators, which fall outside the C0 range, so they must collapse too.
   */
  it("collapses unicode line separators", () => {
    for (const codePoint of [0x0085, 0x2028, 0x2029]) {
      const separator = String.fromCodePoint(codePoint);
      expect(escapeSlackText(`Acme${separator}Payment recovered`)).toBe("Acme Payment recovered");
    }
  });

  it("bounds the rendered length so a value cannot push the alert out of view", () => {
    const escaped = escapeSlackText("a".repeat(500));

    expect(escaped).toBe(`${"a".repeat(256)}…`);
  });

  /**
   * Truncating after escaping would slice an entity in half and emit a dangling `&am`, which
   * renders as literal garbage in the channel. The bound therefore applies to the raw value.
   */
  it("never truncates in the middle of an escaped entity", () => {
    const escaped = escapeSlackText(`${"a".repeat(254)}&&&&`);

    expect(escaped).toBe(`${"a".repeat(254)}&amp;&amp;…`);
    expect(escaped).not.toMatch(/&[a-z]*$/);
  });

  it("counts astral characters as one code point so surrogate pairs stay intact", () => {
    const escaped = escapeSlackText("🔑".repeat(300));

    expect(Array.from(escaped)).toHaveLength(257);
    expect(escaped).not.toContain("�");
  });

  it("leaves ordinary text untouched", () => {
    expect(escapeSlackText("Acme Inc.")).toBe("Acme Inc.");
  });

  it("returns an empty string for missing values", () => {
    expect(escapeSlackText(undefined)).toBe("");
  });
});

/**
 * Guards ENG-3020 at the boundary: the mrkdwn tag escapes every interpolated value, so a new
 * alert cannot embed a customer value unescaped just by writing a template literal.
 */
describe("mrkdwn", () => {
  it("escapes interpolated values but not the literal copy we wrote", () => {
    const name = "<https://evil.example|click>";

    expect(mrkdwn`:tada: Welcome ${name} <- our own text`).toBe(
      ":tada: Welcome &lt;https://evil.example|click&gt; <- our own text",
    );
  });

  it("coerces non-string values instead of throwing", () => {
    // Some callers interpolate fields taken from an external payload, so the runtime type is
    // not guaranteed to match the declared one.
    const unproven = 42 as unknown as string;

    expect(mrkdwn`Type: ${unproven}`).toBe("Type: 42");
    expect(mrkdwn`Name: ${undefined}`).toBe("Name: ");
  });
});

describe("stripeCustomerUrl", () => {
  it("links to the live dashboard for live-mode customers", () => {
    expect(stripeCustomerUrl("cus_123", true)).toBe(
      "https://dashboard.stripe.com/customers/cus_123",
    );
  });

  it("links under /test for test-mode customers so the link resolves", () => {
    expect(stripeCustomerUrl("cus_123", false)).toBe(
      "https://dashboard.stripe.com/test/customers/cus_123",
    );
  });

  it("encodes the id", () => {
    expect(stripeCustomerUrl("cus a/b", true)).toBe(
      "https://dashboard.stripe.com/customers/cus%20a%2Fb",
    );
  });
});

const fetchMock = vi.fn();

beforeEach(() => {
  fetchMock.mockResolvedValue(new Response(null, { status: 200 }));
  vi.stubGlobal("fetch", fetchMock);
  vi.stubEnv("SLACK_WEBHOOK_CUSTOMERS", "https://hooks.slack.example/services/customers");
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
  vi.restoreAllMocks();
  fetchMock.mockReset();
});

type SentTextObject = { type: string; text: string; verbatim?: boolean };
type SentBlock = {
  type: string;
  text?: SentTextObject;
  fields?: SentTextObject[];
  elements?: Array<{
    type: string;
    url?: string;
    text?: SentTextObject | string;
    verbatim?: boolean;
  }>;
};

const sentBlocks = (): SentBlock[] => {
  expect(fetchMock).toHaveBeenCalledTimes(1);
  return JSON.parse(fetchMock.mock.calls[0][1].body).blocks;
};

/** Every mrkdwn text object in the payload: single-text sections plus every field cell. */
const sentMrkdwnObjects = (): SentTextObject[] => {
  const objects: SentTextObject[] = [];
  for (const block of sentBlocks()) {
    if (block.text?.type === "mrkdwn") {
      objects.push(block.text);
    }
    if (Array.isArray(block.fields)) {
      objects.push(...block.fields);
    }
    for (const element of block.elements ?? []) {
      if (element.type === "mrkdwn" && typeof element.text === "string") {
        objects.push({ type: element.type, text: element.text, verbatim: element.verbatim });
      }
    }
  }
  return objects;
};

/** Every mrkdwn string in the payload (excludes the plain_text header/button labels). */
const sentText = (): string => {
  return sentMrkdwnObjects()
    .map((obj) => obj.text)
    .join("\n");
};

const headerText = (): string => {
  const header = sentBlocks().find((block) => block.type === "header");
  return header?.text?.text ?? "";
};

const fieldTexts = (): string[] => {
  const section = sentBlocks().find((block) => Array.isArray(block.fields));
  return (section?.fields ?? []).map((field) => field.text);
};

const buttonUrl = (): string | undefined => {
  const actions = sentBlocks().find((block) => block.type === "actions");
  return actions?.elements?.[0]?.url;
};

// Every value below is free text an attacker can set (Stripe billing name, signup email),
// so no alert may render them as markup.
const HOSTILE_NAME = "<https://evil.example/invoice|View your invoice>";
const HOSTILE_EMAIL = "<https://evil.example|support>@evil.example";
const HOSTILE_TIER = "<https://evil.example|Pro>";

/**
 * Guards ENG-3020: a customer who sets their billing name to slack link syntax must not get a
 * clickable link rendered in our internal channels, which employees would otherwise read as an
 * alert Unkey itself produced. Every alert that embeds a customer-controlled value belongs in
 * this table.
 */
describe("alerts escape attacker-controlled fields", () => {
  const cases: Array<{ name: string; send: () => Promise<unknown> }> = [
    {
      name: "signup",
      send: () =>
        alertCustomerLifecycle({
          action: "signup",
          name: HOSTILE_NAME,
          email: HOSTILE_EMAIL,
          workspaceId: "ws_test",
          workspaceName: HOSTILE_NAME,
          product: HOSTILE_TIER,
          price: "$25",
          stripeCustomerId: "cus_123",
          livemode: true,
        }),
    },
    {
      name: "upgrade",
      send: () =>
        alertCustomerLifecycle({
          action: "upgrade",
          name: HOSTILE_NAME,
          email: HOSTILE_EMAIL,
          workspaceId: "ws_test",
          workspaceName: HOSTILE_NAME,
          product: HOSTILE_TIER,
          previousProduct: HOSTILE_TIER,
          price: "$25",
        }),
    },
    {
      name: "cancelling",
      send: () =>
        alertCustomerLifecycle({
          action: "cancelling",
          name: HOSTILE_NAME,
          email: HOSTILE_EMAIL,
          workspaceId: "ws_test",
          workspaceName: HOSTILE_NAME,
          product: HOSTILE_TIER,
          price: "$25",
        }),
    },
    {
      name: "cancelled",
      send: () =>
        alertCustomerLifecycle({
          action: "cancelled",
          name: HOSTILE_NAME,
          email: HOSTILE_EMAIL,
          workspaceId: "ws_test",
          workspaceName: HOSTILE_NAME,
        }),
    },
    {
      name: "alertPaymentFailed",
      send: () => alertPaymentFailed({ email: HOSTILE_EMAIL, name: HOSTILE_NAME, amount: 2500 }),
    },
    {
      name: "alertPaymentRecovered",
      send: () => alertPaymentRecovered({ email: HOSTILE_EMAIL, name: HOSTILE_NAME, amount: 2500 }),
    },
  ];

  for (const { name, send } of cases) {
    it(`${name} escapes the attacker's link syntax`, async () => {
      await send();

      const text = sentText();
      expect(text).not.toContain("<https://evil.example");
      expect(text).toContain("&lt;");
    });

    it(`${name} sets verbatim on every mrkdwn text object`, async () => {
      await send();

      const objects = sentMrkdwnObjects();
      expect(objects.length).toBeGreaterThan(0);
      for (const obj of objects) {
        expect(obj.verbatim).toBe(true);
      }
    });
  }

  /**
   * Escaping only sees `<url|text>` syntax. A bare URL has nothing to escape, and Slack
   * auto-links it — so `verbatim: true` on the text objects is what stops an attacker-set
   * billing name of `http://evil.example/rotate` from becoming a live link.
   */
  it("does not let a bare-url billing name become a link", async () => {
    await alertCustomerLifecycle({
      action: "signup",
      name: "http://evil.example/rotate",
      email: "jane@acme.com",
      workspaceId: "ws_123",
      workspaceName: "Acme",
    });

    const text = sentText();
    // The value survives as plain text (nothing to escape) but verbatim keeps it unlinked.
    expect(text).toContain("http://evil.example/rotate");
    expect(text).toContain("ws_123");
    for (const obj of sentMrkdwnObjects()) {
      expect(obj.verbatim).toBe(true);
    }
  });
});

describe("alertCustomerLifecycle", () => {
  it("renders the workspace and plan, customer contact, Stripe link and copyable IDs", async () => {
    const workspaceId = `ws_${crypto.randomUUID()}`;
    const stripeCustomerId = `cus_${crypto.randomUUID()}`;
    await alertCustomerLifecycle({
      action: "signup",
      name: "Jane",
      email: "jane@acme.com",
      workspaceId,
      workspaceName: "Acme",
      product: "Pro",
      price: "$25",
      stripeCustomerId,
      livemode: true,
    });

    expect(headerText()).toBe("New customer signup");
    expect(fieldTexts()).toEqual(["*Workspace*\nAcme", "*Plan*\nPro · $25"]);
    expect(sentText()).toContain("*Customer*\nJane\njane@acme.com");
    expect(sentText()).toContain(`Workspace ID: \`${workspaceId}\``);
    expect(sentText()).toContain(`Stripe customer: \`${stripeCustomerId}\``);
    expect(sentText()).toContain(
      `<https://dashboard.stripe.com/customers/${stripeCustomerId}|View customer in Stripe>`,
    );
    expect(JSON.parse(fetchMock.mock.calls[0][1].body).text).toBe(
      "New customer signup. Customer: Jane (jane@acme.com). Workspace: Acme. Plan: Pro · $25.",
    );
  });

  it("shows the tier transition for an upgrade", async () => {
    await alertCustomerLifecycle({
      action: "upgrade",
      name: "Jane",
      email: "jane@acme.com",
      workspaceId: "ws_123",
      workspaceName: "Acme",
      product: "Pro",
      previousProduct: "Free",
      price: "$25",
    });

    expect(headerText()).toBe("Subscription upgraded");
    expect(fieldTexts()).toContain("*Plan*\nFree → Pro · $25");
  });

  it("shows the downgrade and its plan transition", async () => {
    await alertCustomerLifecycle({
      action: "downgrade",
      name: "Jane",
      email: "jane@acme.com",
      workspaceId: "ws_123",
      workspaceName: "Acme",
      product: "Free",
      previousProduct: "Pro",
    });

    expect(headerText()).toBe("Subscription downgraded");
    expect(fieldTexts()).toContain("*Plan*\nPro → Free");
  });

  it("adds a standing note for a cancellation", async () => {
    await alertCustomerLifecycle({
      action: "cancelling",
      name: "Jane",
      email: "jane@acme.com",
      workspaceId: "ws_123",
      workspaceName: "Acme",
      product: "Pro",
      price: "$25",
    });

    expect(headerText()).toBe("Subscription cancelling");
    expect(sentText()).toContain("Worth reaching out to learn why.");
  });

  it("links to the test dashboard when the customer is not live", async () => {
    await alertCustomerLifecycle({
      action: "signup",
      name: "Jane",
      email: "jane@acme.com",
      workspaceId: "ws_123",
      workspaceName: "Acme",
      stripeCustomerId: "cus_123",
      livemode: false,
    });

    expect(sentText()).toContain(
      "<https://dashboard.stripe.com/test/customers/cus_123|View customer in Stripe>",
    );
  });

  it("omits the workspace fields and the Stripe link when they are not known", async () => {
    // A created event can race ahead of the row that links the subscription to its workspace.
    await alertCustomerLifecycle({
      action: "signup",
      name: "Jane",
      email: "jane@acme.com",
      product: "Compute Pro",
      price: "$25",
    });

    expect(fieldTexts()).toEqual(["*Plan*\nCompute Pro · $25"]);
    expect(sentText()).toContain("*Customer*\nJane\njane@acme.com");
    expect(sentText()).not.toMatch(/Workspace|Stripe|undefined/);
  });

  it("keeps the customer contact without creating an empty facts section", async () => {
    await alertCustomerLifecycle({ action: "signup", name: "Jane", email: "jane@acme.com" });
    expect(sentText()).toBe("*Customer*\nJane\njane@acme.com");
    expect(sentBlocks().every((block) => block.fields === undefined)).toBe(true);
  });

  it("shows a known price without inventing an unknown plan", async () => {
    await alertCustomerLifecycle({
      action: "update",
      name: "Jane",
      email: "jane@acme.com",
      price: "$25",
    });
    expect(fieldTexts()).toEqual(["*Price*\n$25"]);
    expect(JSON.parse(fetchMock.mock.calls[0][1].body).text).toBe(
      "Subscription updated. Customer: Jane (jane@acme.com). Price: $25.",
    );
  });

  it("bounds combined contact and plan text within Slack limits", async () => {
    const long = "&".repeat(500);
    await alertCustomerLifecycle({
      action: "upgrade",
      name: long,
      email: long,
      workspaceName: long,
      previousProduct: long,
      product: long,
      price: long,
    });
    for (const block of sentBlocks()) {
      if (block.text) {
        expect(Array.from(block.text.text).length).toBeLessThanOrEqual(3000);
      }
      for (const field of block.fields ?? []) {
        expect(Array.from(field.text).length).toBeLessThanOrEqual(2000);
      }
    }
    expect(sentText()).toContain("&amp;");
  });
});

describe("payment alerts", () => {
  it.each([
    {
      send: alertPaymentFailed,
      title: "Payment failed",
      amount: 2499,
      rendered: "$24.99",
      livemode: false,
    },
    {
      send: alertPaymentRecovered,
      title: "Payment recovered",
      amount: 5001,
      rendered: "$50.01",
      livemode: true,
    },
  ])(
    "renders $title with the amount, customer contact, Stripe link and copyable ID",
    async ({ send, title, amount, rendered, livemode }) => {
      const stripeCustomerId = `cus_${crypto.randomUUID()}`;
      await send({ email: "jane@acme.com", name: "Jane", amount, stripeCustomerId, livemode });

      expect(headerText()).toBe(title);
      expect(sentText()).toContain(`*Amount*\n${rendered}`);
      expect(sentText()).toContain("*Customer*\nJane\njane@acme.com");
      expect(sentText()).toContain(`Stripe customer: \`${stripeCustomerId}\``);
      expect(sentText()).toContain(
        `<https://dashboard.stripe.com${livemode ? "" : "/test"}/customers/${stripeCustomerId}|View customer in Stripe>`,
      );
      expect(JSON.parse(fetchMock.mock.calls[0][1].body).text).toContain(
        `${title}: ${rendered}. Customer: Jane (jane@acme.com).`,
      );
    },
  );

  it("keeps a zero amount and omits unknown Stripe details", async () => {
    await alertPaymentFailed({ email: "jane@acme.com", name: "Jane", amount: 0 });
    expect(sentText()).toContain("*Amount*\n$0.00");
    expect(sentText()).not.toMatch(/Stripe|undefined/);
  });
});

describe("billing attention alerts", () => {
  const ids = () => ({
    subscriptionId: `sub_${crypto.randomUUID()}`,
    workspaceId: `ws_${crypto.randomUUID()}`,
    sessionId: `cs_${crypto.randomUUID()}`,
    eventId: `evt_${crypto.randomUUID()}`,
    productId: `prod_${crypto.randomUUID()}`,
  });
  const fallbackText = (): string => JSON.parse(fetchMock.mock.calls[0][1].body).text;

  for (const livemode of [true, false]) {
    it(`renders the orphan consequence, copyable IDs and labeled link (live: ${livemode})`, async () => {
      const details = ids();
      await alertOrphanedDeploySubscription({
        ...details,
        reason: "subscription_conflict",
        livemode,
      });

      expect(headerText()).toBe("Compute subscription needs attention");
      expect(fieldTexts()).toEqual([
        "*Product*\nCompute",
        "*Reason*\nSubscription conflict",
        `*Workspace ID*\n\`${details.workspaceId}\``,
        `*Subscription*\n\`${details.subscriptionId}\``,
      ]);
      expect(sentText()).toContain(`Checkout session: \`${details.sessionId}\``);
      expect(sentText()).toContain(`Stripe event: \`${details.eventId}\``);
      expect(sentText()).toContain("still billing but is not linked to its workspace.");
      expect(sentText()).toContain(
        `<https://dashboard.stripe.com${livemode ? "" : "/test"}/subscriptions/${details.subscriptionId}|Review subscription in Stripe>`,
      );
      expect(fallbackText()).toBe(
        `Compute subscription ${details.subscriptionId} is billing but unlinked. Review it in Stripe and cancel or refund it manually.`,
      );
      expect(buttonUrl()).toBeUndefined();
      expect(sentMrkdwnObjects().every((object) => object.verbatim)).toBe(true);
    });

    it(`renders metadata consequences and encoded resource links (live: ${livemode})`, async () => {
      const details = { ...ids(), productId: `prod_${crypto.randomUUID()}/a b` };
      await alertInvalidProductQuotaMetadata({ ...details, productName: "Compute Pro", livemode });

      const base = `https://dashboard.stripe.com${livemode ? "" : "/test"}`;
      expect(headerText()).toBe("Product quota settings need attention");
      expect(fieldTexts()).toEqual([
        "*Product*\nCompute Pro",
        `*Subscription*\n\`${details.subscriptionId}\``,
      ]);
      expect(sentText()).toContain(
        `<${base}/products/${encodeURIComponent(details.productId)}|Review product in Stripe> · <${base}/subscriptions/${details.subscriptionId}|View subscription>`,
      );
      expect(sentText()).toContain(`Product ID: \`${details.productId}\``);
      expect(sentText()).toContain(`Stripe event: \`${details.eventId}\``);
      expect(sentText()).toContain("Workspace tier sync was skipped.");
      expect(fallbackText()).toBe(
        "Compute Pro: workspace tier sync was skipped because quota metadata is invalid or missing. Fix the product metadata in Stripe.",
      );
      expect(buttonUrl()).toBeUndefined();
      expect(sentMrkdwnObjects().every((object) => object.verbatim)).toBe(true);
    });
  }

  it("omits unknown optional IDs and retains unknown reasons and API product", async () => {
    await alertOrphanedDeploySubscription({
      subscriptionId: ids().subscriptionId,
      reason: "future_reason",
      product: "API",
    });
    expect(headerText()).toBe("API subscription needs attention");
    expect(fieldTexts()).toContain("*Reason*\nfuture_reason");
    expect(sentText()).not.toMatch(/Workspace ID|Checkout session|Stripe event|undefined/);
    expect(sentBlocks().some((block) => block.type === "context")).toBe(false);
  });

  const senders = [
    (value: string) => alertOrphanedDeploySubscription({ ...ids(), reason: value }),
    (value: string) => alertInvalidProductQuotaMetadata({ ...ids(), productName: value }),
  ];
  for (const [index, send] of senders.entries()) {
    it(`escapes mentions, injected links and forged lines in billing alert ${index}`, async () => {
      const hostile =
        "<!here> <@U123> <https://evil.example|Click>\nforged\u2028line & http://evil.example";
      await send(hostile);
      expect(sentText()).toContain(escapeSlackText(hostile));
      expect(sentText()).not.toContain("<!here>");
      expect(sentText()).not.toContain("<@U123>");
      expect(sentText()).not.toContain("<https://evil.example");
      expect(sentText()).not.toContain("\nforged");
      expect(sentMrkdwnObjects().every((object) => object.verbatim)).toBe(true);
      expect(fallbackText()).not.toMatch(/<!|<@|<https:\/\/evil|\n/);
    });

    it(`bounds free text in billing alert ${index}`, async () => {
      await send("a".repeat(500));
      expect(sentText()).toContain(`${"a".repeat(256)}…`);
      expect(sentText()).not.toContain("a".repeat(257));
    });

    it(`keeps billing alert ${index} best-effort without exposing customer data in logs`, async () => {
      const error = vi.spyOn(console, "error").mockImplementation(() => {});
      vi.spyOn(console, "warn").mockImplementation(() => {});
      fetchMock.mockRejectedValueOnce(new Error("offline"));
      await expect(send("private billing value")).resolves.toBeUndefined();
      fetchMock.mockResolvedValueOnce(new Response(null, { status: 500 }));
      await expect(send("private billing value")).resolves.toBeUndefined();
      expect(error).toHaveBeenCalledTimes(2);
      expect(JSON.stringify(error.mock.calls)).not.toContain("private billing value");
      fetchMock.mockClear();
      vi.stubEnv("SLACK_WEBHOOK_CUSTOMERS", "");
      await expect(send("private billing value")).resolves.toBeUndefined();
      expect(fetchMock).not.toHaveBeenCalled();
    });
  }
});

describe("postToSlack", () => {
  it("does not post when the webhook is not configured", async () => {
    vi.stubEnv("SLACK_WEBHOOK_CUSTOMERS", "");

    const status = await alertCustomerLifecycle({
      action: "cancelled",
      name: "Jane",
      email: "jane@acme.com",
    });

    expect(fetchMock).not.toHaveBeenCalled();
    expect(status).toBe("not_configured");
  });

  /**
   * An alert is best-effort. A Slack outage must not fail the stripe webhook that triggered
   * it, or stripe retries the event and we double-process the subscription.
   */
  it("swallows transport failures", async () => {
    fetchMock.mockRejectedValue(new Error("slack is down"));

    await expect(
      alertCustomerLifecycle({ action: "cancelled", name: "Jane", email: "jane@acme.com" }),
    ).resolves.toBe("failed");
  });

  /**
   * Consolidating the alert callers into postToSlack must not widen where customer data lands:
   * the failure log identifies the alert, not the customer whose event triggered it.
   */
  it("identifies the alert without customer data when slack rejects it", async () => {
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
    fetchMock.mockResolvedValue(new Response(null, { status: 500, statusText: "Server Error" }));

    await alertPaymentFailed({ email: "jane@acme.com", name: "Jane", amount: 2500 });

    expect(consoleError).toHaveBeenCalledTimes(1);
    const logged = JSON.stringify(consoleError.mock.calls[0]);
    expect(logged).not.toContain("jane@acme.com");
    expect(logged).toContain("payment_failed");
  });
});
