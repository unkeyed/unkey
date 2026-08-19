import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  alertCustomerLifecycle,
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
  elements?: Array<{ type: string; url?: string; text?: SentTextObject }>;
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
      // The attacker's `<`/`>` become entities, so no literal link delimiters survive.
      expect(text).not.toContain("<");
      expect(text).not.toContain(">");
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
  it("renders a header, the customer/workspace/plan fields, and a Stripe link", async () => {
    await alertCustomerLifecycle({
      action: "signup",
      name: "Jane",
      email: "jane@acme.com",
      workspaceId: "ws_123",
      workspaceName: "Acme",
      product: "Pro",
      price: "$25",
      stripeCustomerId: "cus_123",
      livemode: true,
    });

    expect(headerText()).toBe(":bugeyes: New customer signup");
    expect(fieldTexts()).toEqual([
      "*Customer*\nJane",
      "*Email*\njane@acme.com",
      "*Workspace*\nAcme",
      "*Workspace ID*\nws_123",
      "*Tier / Product*\nPro",
      "*Price*\n$25",
    ]);
    expect(buttonUrl()).toBe("https://dashboard.stripe.com/customers/cus_123");
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

    expect(headerText()).toBe(":stonks: Subscription upgraded");
    expect(fieldTexts()).toContain("*Tier / Product*\nFree → Pro");
  });

  it("marks a downgrade with its own emoji", async () => {
    await alertCustomerLifecycle({
      action: "downgrade",
      name: "Jane",
      email: "jane@acme.com",
      workspaceId: "ws_123",
      workspaceName: "Acme",
      product: "Free",
      previousProduct: "Pro",
    });

    expect(headerText()).toBe(":notstonks: Subscription downgraded");
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

    expect(headerText()).toBe(":warning: Subscription cancelling");
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

    expect(buttonUrl()).toBe("https://dashboard.stripe.com/test/customers/cus_123");
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

    expect(fieldTexts()).toEqual([
      "*Customer*\nJane",
      "*Email*\njane@acme.com",
      "*Tier / Product*\nCompute Pro",
      "*Price*\n$25",
    ]);
    expect(buttonUrl()).toBeUndefined();
  });
});

describe("alertPaymentFailed", () => {
  it("renders the amount and a Stripe link", async () => {
    await alertPaymentFailed({
      email: "jane@acme.com",
      name: "Jane",
      amount: 2500,
      stripeCustomerId: "cus_123",
      livemode: true,
    });

    expect(headerText()).toBe(":warning: Payment failed");
    expect(fieldTexts()).toEqual([
      "*Customer*\nJane",
      "*Email*\njane@acme.com",
      "*Amount*\n$25.00",
    ]);
    expect(buttonUrl()).toBe("https://dashboard.stripe.com/customers/cus_123");
  });
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
