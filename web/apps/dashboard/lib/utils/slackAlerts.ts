import { formatPrice } from "@/lib/fmt";

/**
 * The longest a value may render as. A billing name is free text and a leaked-key URL carries
 * attacker-chosen path segments, so without a bound either can push the rest of the alert out
 * of view in the channel.
 */
const RENDERED_LENGTH_MAX = 256;

/** Bounds a string to RENDERED_LENGTH_MAX code points, slicing by code point so surrogate pairs stay whole. */
function boundLength(value: string): string {
  const codePoints = Array.from(value);
  if (codePoints.length <= RENDERED_LENGTH_MAX) {
    return value;
  }
  return `${codePoints.slice(0, RENDERED_LENGTH_MAX).join("")}…`;
}

/**
 * Escapes a customer-controlled value for use inside a Slack mrkdwn field.
 *
 * Slack parses `<...>` as a link and `&` as an entity, so an unescaped billing name such as
 * `<https://evil.example|View your invoice>` renders as a live link in our internal channels
 * and reads as though Unkey sent it.
 *
 * Line breaks collapse to spaces, including the Unicode separators U+0085, U+2028 and U+2029
 * that some renderers honour: Slack renders each line of a mrkdwn section separately, so a
 * value containing a break can otherwise forge what looks like an additional status line
 * emitted by the system.
 *
 * Slack offers no escape for its formatting characters (`*`, `_`, `~`, backtick), so an
 * untrusted value can still render as bold or italic. That is cosmetic. It cannot forge a
 * link or a line, which are the vectors that matter here.
 *
 * https://docs.slack.dev/messaging/formatting-message-text#escaping
 */
export function escapeSlackText(value: string | undefined): string {
  if (!value) {
    return "";
  }

  const collapsed = value
    // biome-ignore lint/suspicious/noControlCharactersInRegex: collapsing them is the point
    .replace(/[\u0000-\u001f\u007f\u0085\u2028\u2029]+/g, " ")
    .trim();

  // Bound before escaping so the slice cannot cut an entity such as `&amp;` in half.
  return boundLength(collapsed)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;");
}

/**
 * A value allowed into a mrkdwn field without escaping. It is deliberately not exported and
 * has exactly one producer, `slackUrl`, so escaping is the only way to get a value in from
 * outside this module.
 */
class TrustedSlackText {
  constructor(readonly value: string) {}
}

/**
 * Renders a URL so that it stays clickable in Slack.
 *
 * Escaping a URL outright would turn a `&` in its query string into `&amp;` and break the
 * link, but rendering one raw is not safe either: arriving over a verified channel does not
 * make a URL's contents trustworthy. GitHub secret scanning reports the URL of the file
 * holding the leaked key, and the attacker picks their own repository, branch and file names,
 * which git allows to contain `<`, `>` and `|`, the characters Slack reads as link syntax.
 *
 * Normalizing through `URL` percent-encodes those characters and leaves the query string
 * intact. A value that is not a parseable http(s) URL is returned untrusted, so `mrkdwn`
 * escapes it as ordinary text: a malformed URL degrades to unclickable, not to injectable.
 * Only pass the result to `mrkdwn`, which is what escapes the untrusted case.
 */
export function slackUrl(value: string): TrustedSlackText | string {
  let parsed: URL;
  try {
    parsed = new URL(value);
  } catch {
    return value;
  }

  if (parsed.protocol !== "https:" && parsed.protocol !== "http:") {
    return value;
  }
  return new TrustedSlackText(boundLength(parsed.href));
}

/**
 * A string that has been through the `mrkdwn` tag. Blocks accept nothing else, so a plain
 * template literal carrying an unescaped customer value fails to type-check rather than
 * silently reopening ENG-3020.
 */
type Mrkdwn = string & { readonly __mrkdwn: unique symbol };

type SlackValue = string | number | undefined | TrustedSlackText;

/**
 * Builds a mrkdwn string, escaping every interpolated value that is not explicitly trusted.
 *
 * This is the boundary that keeps ENG-3020 fixed. Escaping is the default rather than
 * something each author must remember at each `${}`, and the only way past it is `slackUrl`,
 * which validates what it lets through. Values are coerced with `String` first, because some
 * callers interpolate fields parsed from an external payload whose runtime type is unproven.
 */
export function mrkdwn(strings: TemplateStringsArray, ...values: SlackValue[]): Mrkdwn {
  const text = strings.reduce((acc, literal, index) => {
    if (index === 0) {
      return literal;
    }

    const value = values[index - 1];
    if (value instanceof TrustedSlackText) {
      return `${acc}${value.value}${literal}`;
    }
    const rendered = value === undefined ? "" : escapeSlackText(String(value));
    return `${acc}${rendered}${literal}`;
  }, "");

  // The brand exists only in the type system, and this is the sole place that mints it.
  return text as Mrkdwn;
}

type SlackBlock = {
  type: "section";
  text?: { type: "mrkdwn"; text: Mrkdwn };
  fields?: Array<{ type: "mrkdwn"; text: Mrkdwn }>;
};

type SlackMessage = {
  text?: string;
  blocks: SlackBlock[];
};

/**
 * Posts a message to a Slack incoming webhook.
 *
 * Alerts are best-effort: a failure here must never fail the stripe webhook or the GitHub
 * secret-scanning response that triggered it, so delivery problems are logged rather than
 * thrown. The log identifies the alert only, so that consolidating the seven callers here
 * does not spread their customer data any wider than the alert itself.
 */
async function postToSlack(
  webhookUrl: string | undefined,
  alert: string,
  message: SlackMessage,
): Promise<void> {
  if (!webhookUrl) {
    console.warn(`Slack webhook is not configured, skipping alert: ${alert}`);
    return;
  }

  try {
    const response = await fetch(webhookUrl, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(message),
    });

    if (!response.ok) {
      console.error("Failed to send Slack alert:", {
        alert,
        status: response.status,
        statusText: response.statusText,
      });
    }
  } catch (err: unknown) {
    console.error("Error sending Slack alert:", {
      alert,
      error: err instanceof Error ? { name: err.name, message: err.message } : "unknown error",
    });
  }
}

const section = (text: Mrkdwn): SlackBlock => {
  return { type: "section", text: { type: "mrkdwn", text } };
};

type LeakedKeyProps = {
  type: string;
  source: string;
  itemUrl: string;
  date: string;
  keyId: string;
  workspaceName: string;
  orgId: string;
  email: string;
};

/** Posts a leaked-key alert from GitHub secret scanning to the on-call channel. */
export async function alertLeakedKey({
  type,
  source,
  itemUrl,
  date,
  keyId,
  workspaceName,
  orgId,
  email,
}: LeakedKeyProps): Promise<void> {
  await postToSlack(process.env.SLACK_WEBHOOK_URL_LEAKED_KEY, "leaked_key", {
    text: "Leaked Key Found",
    blocks: [
      {
        type: "section",
        fields: [
          {
            type: "mrkdwn",
            text: mrkdwn`Type: ${type} \n Source: ${source} \n Date: ${date} \n URL: ${slackUrl(itemUrl)}`,
          },
          {
            type: "mrkdwn",
            text: mrkdwn`Key: ${keyId} \n Workspace: ${workspaceName} \n Tenant: ${orgId} \n User: ${email}`,
          },
        ],
      },
    ],
  });
}

export async function alertSubscriptionCreation(
  product: string,
  price: string,
  email: string,
  name?: string,
): Promise<void> {
  await postToSlack(process.env.SLACK_WEBHOOK_CUSTOMERS, "subscription_created", {
    blocks: [
      section(mrkdwn`:bugeyes: New customer ${name} signed up`),
      section(
        mrkdwn`A new subscription for the ${product} tier has started at a price of ${price} by ${email} :moneybag: `,
      ),
    ],
  });
}

export async function alertSubscriptionUpdate(
  product: string,
  price: string,
  email: string,
  name?: string,
  changeType?: string,
  previousTier?: string,
): Promise<void> {
  let emoji = ":stonks:";
  let actionText = "updated their subscription";

  if (changeType === "upgraded") {
    actionText = "upgraded their subscription";
  } else if (changeType === "downgraded") {
    emoji = ":notstonks:";
    actionText = "downgraded their subscription";
  }

  let subscriptionText = mrkdwn`Subscription ${changeType} to the ${product} tier`;
  if (previousTier && changeType !== "updated") {
    subscriptionText = mrkdwn`${name}'s subscription ${changeType} from ${previousTier} to ${product} tier, they are now paying ${price}. `;
  }

  await postToSlack(process.env.SLACK_WEBHOOK_CUSTOMERS, "subscription_updated", {
    blocks: [
      // emoji and actionText are our own copy, so escaping them is a no-op. They go through
      // the tag anyway rather than bypassing it, because nothing here is worth an exemption.
      section(mrkdwn`${emoji} ${name} ${actionText}`),
      section(subscriptionText),
      section(mrkdwn`Here is their contact information: ${email}`),
    ],
  });
}

export async function alertIsCancellingSubscription(
  product: string,
  price: string,
  email: string,
  name?: string,
): Promise<void> {
  await postToSlack(process.env.SLACK_WEBHOOK_CUSTOMERS, "subscription_cancelling", {
    blocks: [
      section(mrkdwn`:warning: ${name} is cancelling their subscription.`),
      section(
        mrkdwn`Subscription cancellation requested by ${email} - for ${product} at ${price} they will be moved back to the free tier, at the end of the month. We should reach out to find out why they are cancelling.`,
      ),
    ],
  });
}

export async function alertSubscriptionCancelled(email: string, name?: string): Promise<void> {
  await postToSlack(process.env.SLACK_WEBHOOK_CUSTOMERS, "subscription_cancelled", {
    blocks: [
      section(mrkdwn`:caleb-sad: ${name} cancelled their subscription`),
      section(
        mrkdwn`Subscription cancelled by ${email} - they've been moved back to the free tier`,
      ),
    ],
  });
}

export async function alertPaymentFailed(
  customerEmail: string,
  customerName: string,
  amount: number,
): Promise<void> {
  const formattedAmount = formatPrice(amount);

  await postToSlack(process.env.SLACK_WEBHOOK_CUSTOMERS, "payment_failed", {
    blocks: [
      section(mrkdwn`:warning: Payment failed for ${customerName}`),
      section(
        mrkdwn`Payment of ${formattedAmount} failed for ${customerEmail}. We should reach out to help resolve the payment issue.`,
      ),
    ],
  });
}

export async function alertPaymentRecovered(
  customerEmail: string,
  customerName: string,
  amount: number,
): Promise<void> {
  const formattedAmount = formatPrice(amount);

  await postToSlack(process.env.SLACK_WEBHOOK_CUSTOMERS, "payment_recovered", {
    blocks: [
      section(mrkdwn`:tada: Payment recovered for ${customerName}`),
      section(
        mrkdwn`Great news! Payment of ${formattedAmount} has been successfully processed for ${customerEmail} after a previous failure. Their service should now be restored.`,
      ),
    ],
  });
}

/**
 * High-severity ops alert for a paid Compute checkout whose subscription could
 * not be linked to its workspace and cannot be reconciled automatically — most
 * importantly a `subscription_conflict`, where a race created a second live,
 * charged subscription while the workspace is already served by another. The
 * subscription keeps billing until an operator cancels/refunds it, so this must
 * page a human rather than sit in logs.
 */
export async function alertOrphanedDeploySubscription(details: {
  workspaceId: string;
  subscriptionId: string;
  sessionId: string;
  reason: string;
}): Promise<void> {
  const url = process.env.SLACK_WEBHOOK_CUSTOMERS;
  if (!url) {
    return;
  }

  try {
    const response = await fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        blocks: [
          {
            type: "section",
            text: {
              type: "mrkdwn",
              text: `:rotating_light: Orphaned Compute subscription needs manual reconciliation (${details.reason})`,
            },
          },
          {
            type: "section",
            text: {
              type: "mrkdwn",
              text: `A paid Compute checkout could not be linked to its workspace. The subscription is billing but unlinked — cancel/refund it manually.\nWorkspace: \`${details.workspaceId}\`\nSubscription: \`${details.subscriptionId}\`\nCheckout session: \`${details.sessionId}\``,
            },
          },
        ],
      }),
    });

    if (!response.ok) {
      console.error("Failed to send orphaned-subscription alert to Slack:", {
        status: response.status,
        statusText: response.statusText,
        ...details,
      });
    }
  } catch (err: unknown) {
    console.error("Error sending orphaned-subscription alert:", {
      error: err instanceof Error ? { message: err.message, name: err.name } : err,
      ...details,
    });
  }
}
