import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  alertIsCancellingSubscription,
  alertLeakedKey,
  alertPaymentFailed,
  alertPaymentRecovered,
  alertSubscriptionCancelled,
  alertSubscriptionCreation,
  alertSubscriptionUpdate,
  escapeSlackText,
  mrkdwn,
  slackUrl,
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

  /**
   * `slackUrl` is the module's only producer of trusted text, so it is the only way an
   * interpolated value can reach a mrkdwn field without being escaped.
   */
  it("renders trusted values raw", () => {
    expect(mrkdwn`URL: ${slackUrl("https://github.com/acme/api")}`).toBe(
      "URL: https://github.com/acme/api",
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

/**
 * The leaked-key URL is the one field rendered without escaping, so that a query string stays
 * clickable. It is also attacker-influenced: GitHub reports the URL of the file holding the
 * key, and the attacker names their own repo, branch and file. It must not be able to carry
 * slack link syntax.
 */
describe("slackUrl", () => {
  it("keeps a query string clickable rather than escaping the ampersand", () => {
    expect(mrkdwn`URL: ${slackUrl("https://github.com/acme/api?ref=abc&line=10")}`).toBe(
      "URL: https://github.com/acme/api?ref=abc&line=10",
    );
  });

  it("percent-encodes link syntax smuggled through a repo or file name", () => {
    const hostile = "https://github.com/acme/<https://evil.example|Rotate your key now>.ts";

    const rendered = mrkdwn`URL: ${slackUrl(hostile)}`;

    expect(rendered).not.toContain("<");
    expect(rendered).not.toContain(">");
    expect(rendered).toContain("%3C");
    expect(rendered).toContain("%3E");
  });

  it("falls back to escaped text when the value is not an http url", () => {
    expect(mrkdwn`URL: ${slackUrl("javascript:alert(1)")}`).toBe("URL: javascript:alert(1)");
    expect(mrkdwn`URL: ${slackUrl("<not a url>")}`).toBe("URL: &lt;not a url&gt;");
  });

  /**
   * The URL is the one field that skips escaping, so it must not skip the length bound too: a
   * repo/branch/path GitHub reports can be kilobytes long and would otherwise bury the second
   * field with the key and workspace an on-call engineer needs.
   */
  it("bounds an adversarially long url", () => {
    const longUrl = `https://github.com/acme/${"a".repeat(500)}`;

    const rendered = mrkdwn`${slackUrl(longUrl)}`;

    expect(Array.from(rendered)).toHaveLength(257);
    expect(rendered.endsWith("…")).toBe(true);
  });
});

const fetchMock = vi.fn();

beforeEach(() => {
  fetchMock.mockResolvedValue(new Response(null, { status: 200 }));
  vi.stubGlobal("fetch", fetchMock);
  vi.stubEnv("SLACK_WEBHOOK_CUSTOMERS", "https://hooks.slack.example/services/customers");
  vi.stubEnv("SLACK_WEBHOOK_URL_LEAKED_KEY", "https://hooks.slack.example/services/leaked");
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
  vi.restoreAllMocks();
  fetchMock.mockReset();
});

const sentBlocks = (): Array<{ text?: { text: string }; fields?: Array<{ text: string }> }> => {
  expect(fetchMock).toHaveBeenCalledTimes(1);
  return JSON.parse(fetchMock.mock.calls[0][1].body).blocks;
};

/** Every mrkdwn string in the posted payload, regardless of whether the block uses text or fields. */
const sentText = (): string => {
  return sentBlocks()
    .flatMap((block) => (block.fields ? block.fields.map((f) => f.text) : [block.text?.text ?? ""]))
    .join("\n");
};

// Every value below is free text an attacker can set (Stripe billing name, signup email,
// workspace name, and the repo path GitHub reports), so no alert may render them as markup.
const HOSTILE_NAME = "<https://evil.example/invoice|View your invoice>";
const HOSTILE_EMAIL = "<https://evil.example|support>@evil.example";
const HOSTILE_TIER = "<https://evil.example|Pro>";
const HOSTILE_URL = "https://github.com/acme/<https://evil.example|Rotate your key now>.ts";

/**
 * Guards ENG-3020: a customer who sets their billing name or workspace name to slack link
 * syntax must not get a clickable link rendered in our internal channels, which employees
 * would otherwise read as an alert Unkey itself produced. Every alert that embeds a
 * customer-controlled value belongs in this table.
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
    {
      name: "alertLeakedKey",
      send: () =>
        alertLeakedKey({
          type: "unkey_root_key",
          source: "commit",
          itemUrl: HOSTILE_URL,
          date: "Mon Jul 14 2026",
          keyId: "key_123",
          workspaceName: HOSTILE_NAME,
          orgId: "org_123",
          email: HOSTILE_EMAIL,
        }),
    },
  ];

  for (const { name, send } of cases) {
    it(`${name} sends no unescaped angle brackets`, async () => {
      await send();

      const text = sentText();
      expect(text).not.toContain("<");
      expect(text).not.toContain(">");
    });
  }
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

describe("alertLeakedKey", () => {
  it("leaves the github-supplied url raw so it stays clickable", async () => {
    await alertLeakedKey({
      type: "unkey_root_key",
      source: "commit",
      itemUrl: "https://github.com/acme/api/blob/main/index.ts?ref=abc&line=10",
      date: "Mon Jul 14 2026",
      keyId: "key_123",
      workspaceName: "Acme",
      orgId: "org_123",
      email: "jane@acme.com",
    });

    const fields = sentBlocks()[0].fields ?? [];
    expect(fields[0].text).toContain(
      "URL: https://github.com/acme/api/blob/main/index.ts?ref=abc&line=10",
    );
    expect(fields[1].text).toBe(
      "Key: key_123 \n Workspace: Acme \n Tenant: org_123 \n User: jane@acme.com",
    );
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
