import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  alertIsCancellingSubscription,
  alertPaymentFailed,
  alertPaymentRecovered,
  alertSubscriptionCancelled,
  alertSubscriptionCreation,
  alertSubscriptionUpdate,
  escapeSlackText,
  mrkdwn,
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
 * Guards ENG-3020 at the boundary: the mrkdwn tag is what makes escaping the default, so a
 * new alert cannot embed a customer value unescaped without explicitly marking it trusted.
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

const sentBlocks = (): Array<{ text: SentTextObject }> => {
  expect(fetchMock).toHaveBeenCalledTimes(1);
  return JSON.parse(fetchMock.mock.calls[0][1].body).blocks;
};

/** Every mrkdwn text object in the posted payload. */
const sentTextObjects = (): SentTextObject[] => {
  return sentBlocks().map((block) => block.text);
};

/** Every mrkdwn string in the posted payload. */
const sentText = (): string => {
  return sentTextObjects()
    .map((obj) => obj.text)
    .join("\n");
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
  const cases: Array<{ name: string; send: () => Promise<void> }> = [
    {
      name: "alertSubscriptionCreation",
      send: () => alertSubscriptionCreation(HOSTILE_TIER, "$25", HOSTILE_EMAIL, HOSTILE_NAME),
    },
    {
      name: "alertSubscriptionUpdate",
      send: () =>
        alertSubscriptionUpdate(
          HOSTILE_TIER,
          "$25",
          HOSTILE_EMAIL,
          HOSTILE_NAME,
          "upgraded",
          HOSTILE_TIER,
        ),
    },
    {
      name: "alertIsCancellingSubscription",
      send: () => alertIsCancellingSubscription(HOSTILE_TIER, "$25", HOSTILE_EMAIL, HOSTILE_NAME),
    },
    {
      name: "alertSubscriptionCancelled",
      send: () => alertSubscriptionCancelled(HOSTILE_EMAIL, HOSTILE_NAME),
    },
    {
      name: "alertPaymentFailed",
      send: () => alertPaymentFailed(HOSTILE_EMAIL, HOSTILE_NAME, 2500),
    },
    {
      name: "alertPaymentRecovered",
      send: () => alertPaymentRecovered(HOSTILE_EMAIL, HOSTILE_NAME, 2500),
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

    it(`${name} sets verbatim on every text object`, async () => {
      await send();

      const objects = sentTextObjects();
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
    await alertSubscriptionCreation("Pro", "$25", "jane@acme.com", "http://evil.example/rotate");

    const text = sentText();
    // The value survives as plain text (nothing to escape) but verbatim keeps it unlinked.
    expect(text).toContain("http://evil.example/rotate");
    for (const obj of sentTextObjects()) {
      expect(obj.verbatim).toBe(true);
    }
  });
});

describe("postToSlack", () => {
  it("does not post when the webhook is not configured", async () => {
    vi.stubEnv("SLACK_WEBHOOK_CUSTOMERS", "");

    await alertSubscriptionCancelled("jane@acme.com", "Jane");

    expect(fetchMock).not.toHaveBeenCalled();
  });

  /**
   * An alert is best-effort. A Slack outage must not fail the stripe webhook that triggered
   * it, or stripe retries the event and we double-process the subscription.
   */
  it("swallows transport failures", async () => {
    fetchMock.mockRejectedValue(new Error("slack is down"));

    await expect(alertSubscriptionCancelled("jane@acme.com", "Jane")).resolves.toBeUndefined();
  });

  /**
   * Consolidating the alert callers into postToSlack must not widen where customer data lands:
   * the failure log identifies the alert, not the customer whose event triggered it.
   */
  it("identifies the alert without customer data when slack rejects it", async () => {
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
    fetchMock.mockResolvedValue(new Response(null, { status: 500, statusText: "Server Error" }));

    await alertPaymentFailed("jane@acme.com", "Jane", 2500);

    expect(consoleError).toHaveBeenCalledTimes(1);
    const logged = JSON.stringify(consoleError.mock.calls[0]);
    expect(logged).not.toContain("jane@acme.com");
    expect(logged).toContain("payment_failed");
  });
});

describe("alertSubscriptionUpdate", () => {
  const blockText = (index: number): string => {
    return sentBlocks()[index].text?.text ?? "";
  };

  it("describes the tier change when a previous tier is known", async () => {
    await alertSubscriptionUpdate("Pro", "$25", "jane@acme.com", "Jane", "upgraded", "Free");

    expect(blockText(0)).toBe(":stonks: Jane upgraded their subscription");
    expect(blockText(1)).toBe(
      "Jane's subscription upgraded from Free to Pro tier, they are now paying $25. ",
    );
    expect(blockText(2)).toBe("Here is their contact information: jane@acme.com");
  });

  it("falls back to the plain message when the previous tier is unknown", async () => {
    // changeType is "upgraded", so only the missing previousTier can select the fallback.
    await alertSubscriptionUpdate("Pro", "$25", "jane@acme.com", "Jane", "upgraded");

    expect(blockText(1)).toBe("Subscription upgraded to the Pro tier");
  });

  it("falls back to the plain message for a plain update", async () => {
    await alertSubscriptionUpdate("Pro", "$25", "jane@acme.com", "Jane", "updated", "Free");

    expect(blockText(0)).toBe(":stonks: Jane updated their subscription");
    expect(blockText(1)).toBe("Subscription updated to the Pro tier");
  });

  it("marks downgrades with a different emoji", async () => {
    await alertSubscriptionUpdate("Free", "$0", "jane@acme.com", "Jane", "downgraded", "Pro");

    expect(blockText(0)).toBe(":notstonks: Jane downgraded their subscription");
  });
});
