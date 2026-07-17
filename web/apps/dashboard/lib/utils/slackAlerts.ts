import { formatPrice } from "@/lib/fmt";

/**
 * The longest a value may render as. A billing name is free text, so without a bound a
 * customer can push the rest of the alert out of view in the channel.
 */
const RENDERED_LENGTH_MAX = 256;

/** Bounds a string to RENDERED_LENGTH_MAX code points, slicing by code point so surrogate pairs stay whole. */
function boundLength(value: string): string {
  // UTF-16 length is an upper bound on the code-point count, so a short string skips the
  // Array.from allocation entirely; only genuinely long values pay for the exact count.
  if (value.length <= RENDERED_LENGTH_MAX) {
    return value;
  }
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
 * and reads as though Unkey sent it. Escaping closes this explicit-link vector; the separate
 * `verbatim: true` on the text objects closes bare-URL auto-linking, which escaping cannot see.
 *
 * Line breaks collapse to spaces, including the Unicode separators U+0085, U+2028 and U+2029
 * that some renderers honour: Slack renders each line of a mrkdwn section separately, so a
 * value containing a break can otherwise forge what looks like an additional status line
 * emitted by the system.
 *
 * Slack offers no escape for its formatting characters (`*`, `_`, `~`, backtick), so an
 * untrusted value can still render as bold or italic. That is cosmetic styling only; it cannot
 * forge a link or a line, which are the vectors that matter here.
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
 * A string that has been through the `mrkdwn` tag. Blocks accept nothing else, so a plain
 * template literal carrying an unescaped customer value fails to type-check rather than
 * silently reopening ENG-3020.
 */
type Mrkdwn = string & { readonly __mrkdwn: unique symbol };

type SlackValue = string | number | undefined;

/**
 * Builds a mrkdwn string, escaping every interpolated value.
 *
 * This is the boundary that keeps ENG-3020 fixed. Escaping is the default rather than
 * something each author must remember at each `${}`. Values are coerced with `String` first,
 * because some callers interpolate fields whose runtime type is unproven.
 */
export function mrkdwn(strings: TemplateStringsArray, ...values: SlackValue[]): Mrkdwn {
  const text = strings.reduce((acc, literal, index) => {
    if (index === 0) {
      return literal;
    }

    const value = values[index - 1];
    const rendered = value === undefined ? "" : escapeSlackText(String(value));
    return `${acc}${rendered}${literal}`;
  }, "");

  // The brand exists only in the type system, and this is the sole place that mints it.
  return text as Mrkdwn;
}

/**
 * A mrkdwn text object with `verbatim: true`, which tells Slack to skip its preprocessing —
 * most importantly, auto-linking bare URLs and emails. Escaping handles the explicit
 * `<url|text>` markup; verbatim handles the bare-URL vector that escaping cannot see.
 */
type MrkdwnText = { type: "mrkdwn"; text: Mrkdwn; verbatim: true };

type SlackBlock = {
  type: "section";
  text: MrkdwnText;
};

const mrkdwnText = (text: Mrkdwn): MrkdwnText => {
  return { type: "mrkdwn", text, verbatim: true };
};

type SlackMessage = {
  blocks: SlackBlock[];
};

/**
 * Posts a message to a Slack incoming webhook.
 *
 * Alerts are best-effort: a failure here must never fail the stripe webhook that triggered
 * it, so delivery problems are logged rather than thrown. The log identifies the alert only,
 * so that consolidating the callers here does not spread their customer data any wider than
 * the alert itself.
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
  return { type: "section", text: mrkdwnText(text) };
};

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
