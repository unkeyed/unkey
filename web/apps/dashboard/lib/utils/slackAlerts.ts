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

/**
 * A plain_text object. Slack renders it literally — no markup, no auto-linking — so it is only
 * ever fed our own copy (block titles, button labels), never a customer-controlled value.
 */
type PlainText = { type: "plain_text"; text: string; emoji: true };

type HeaderBlock = { type: "header"; text: PlainText };

/** A section with a single line of text, `fields` for a two-column grid, or both. */
type SectionBlock = { type: "section"; text?: MrkdwnText; fields?: MrkdwnText[] };

/** A link button. `url` is a separate field from any text, so it carries no injection risk. */
type ButtonElement = { type: "button"; text: PlainText; url: string };
type ActionsBlock = { type: "actions"; elements: ButtonElement[] };

type SlackBlock = HeaderBlock | SectionBlock | ActionsBlock;

const mrkdwnText = (text: Mrkdwn): MrkdwnText => {
  return { type: "mrkdwn", text, verbatim: true };
};

const plainText = (text: string): PlainText => {
  return { type: "plain_text", text, emoji: true };
};

type SlackMessage = {
  blocks: SlackBlock[];
};

export type SlackPostStatus = "sent" | "not_configured" | "rejected" | "failed";

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
): Promise<SlackPostStatus> {
  if (!webhookUrl) {
    console.warn(`Slack webhook is not configured, skipping alert: ${alert}`);
    return "not_configured";
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
      return "rejected";
    }
    return "sent";
  } catch (err: unknown) {
    console.error("Error sending Slack alert:", {
      alert,
      error: err instanceof Error ? { name: err.name, message: err.message } : "unknown error",
    });
    return "failed";
  }
}

const section = (text: Mrkdwn): SectionBlock => {
  return { type: "section", text: mrkdwnText(text) };
};

const header = (text: string): HeaderBlock => {
  return { type: "header", text: plainText(text) };
};

/** A two-column grid of labelled values. Slack caps a section at 10 fields; we send at most 6. */
const fieldsSection = (fields: Mrkdwn[]): SectionBlock => {
  return { type: "section", fields: fields.map(mrkdwnText) };
};

const linkButton = (label: string, url: string): ActionsBlock => {
  return { type: "actions", elements: [{ type: "button", text: plainText(label), url }] };
};

/**
 * A deep link to the customer in the Stripe dashboard. Test-mode objects live under `/test`, so
 * the link must switch on `livemode` or it 404s in one of the two modes. The id is Stripe-issued
 * (`cus_...`), not customer free text, but it is still encoded before going into the URL.
 */
export function stripeCustomerUrl(customerId: string, livemode: boolean): string {
  const base = livemode ? "https://dashboard.stripe.com" : "https://dashboard.stripe.com/test";
  return `${base}/customers/${encodeURIComponent(customerId)}`;
}

/**
 * The subscription-lifecycle action an alert announces. These are the states the team watches in
 * the customers channel; each maps to a header emoji, a title, and an optional standing note.
 */
export type CustomerAlertAction =
  | "signup"
  | "upgrade"
  | "downgrade"
  | "update"
  | "cancelling"
  | "cancelled";

const ACTION_META: Record<CustomerAlertAction, { emoji: string; title: string; note?: Mrkdwn }> = {
  signup: { emoji: ":bugeyes:", title: "New customer signup" },
  upgrade: { emoji: ":stonks:", title: "Subscription upgraded" },
  downgrade: { emoji: ":notstonks:", title: "Subscription downgraded" },
  update: { emoji: ":stonks:", title: "Subscription updated" },
  cancelling: {
    emoji: ":warning:",
    title: "Subscription cancelling",
    // note is our own copy; it goes through the tag so the type checks, escaping is a no-op.
    note: mrkdwn`They stay on their plan until the end of the billing period, then move back to the free tier. Worth reaching out to learn why.`,
  },
  cancelled: {
    emoji: ":caleb-sad:",
    title: "Subscription cancelled",
    note: mrkdwn`They've been moved back to the free tier.`,
  },
};

/**
 * The details a subscription-lifecycle alert renders. Every string field is customer-controlled
 * free text except the ids and `action`; the `mrkdwn` tag escapes them all at the boundary.
 * Workspace fields are optional because a `created` event can race ahead of the row that links
 * the subscription to its workspace, leaving nothing to resolve the name/id from yet.
 */
export type CustomerLifecycleAlert = {
  action: CustomerAlertAction;
  name: string;
  email: string;
  workspaceId?: string;
  workspaceName?: string;
  product?: string;
  previousProduct?: string;
  price?: string;
  stripeCustomerId?: string;
  livemode?: boolean;
};

/**
 * Posts the Block Kit alert for a subscription-lifecycle event: a header naming the action, a
 * two-column grid of the customer/workspace/plan facts the team acts on, an optional standing
 * note, and a button deep-linking to the customer in Stripe. Fields that are absent (e.g. no
 * price on a cancellation) are simply omitted so the grid never shows a blank cell.
 */
export async function alertCustomerLifecycle(
  alert: CustomerLifecycleAlert,
): Promise<SlackPostStatus> {
  const meta = ACTION_META[alert.action];

  const fields: Mrkdwn[] = [mrkdwn`*Customer*\n${alert.name}`, mrkdwn`*Email*\n${alert.email}`];
  if (alert.workspaceName) {
    fields.push(mrkdwn`*Workspace*\n${alert.workspaceName}`);
  }
  if (alert.workspaceId) {
    fields.push(mrkdwn`*Workspace ID*\n${alert.workspaceId}`);
  }
  if (alert.product) {
    fields.push(
      alert.previousProduct
        ? mrkdwn`*Tier / Product*\n${alert.previousProduct} → ${alert.product}`
        : mrkdwn`*Tier / Product*\n${alert.product}`,
    );
  }
  if (alert.price) {
    fields.push(mrkdwn`*Price*\n${alert.price}`);
  }

  const blocks: SlackBlock[] = [header(`${meta.emoji} ${meta.title}`), fieldsSection(fields)];
  if (meta.note) {
    blocks.push(section(meta.note));
  }
  if (alert.stripeCustomerId) {
    blocks.push(
      linkButton(
        "View customer in Stripe",
        stripeCustomerUrl(alert.stripeCustomerId, alert.livemode ?? true),
      ),
    );
  }

  return postToSlack(process.env.SLACK_WEBHOOK_CUSTOMERS, `subscription_${alert.action}`, {
    blocks,
  });
}

/**
 * The details a payment alert renders. Payment events arrive on invoices, which are not linked to
 * a workspace here, so these carry only the customer facts plus the Stripe deep link.
 */
export type PaymentAlert = {
  email: string;
  name: string;
  amount: number;
  stripeCustomerId?: string;
  livemode?: boolean;
};

/** Shared renderer for the two payment alerts: header, customer/amount grid, note, Stripe link. */
async function alertPayment(
  alert: PaymentAlert,
  kind: "failed" | "recovered",
  emoji: string,
  title: string,
  note: Mrkdwn,
): Promise<void> {
  const blocks: SlackBlock[] = [
    header(`${emoji} ${title}`),
    fieldsSection([
      mrkdwn`*Customer*\n${alert.name}`,
      mrkdwn`*Email*\n${alert.email}`,
      // formatPrice renders USD; the invoice currency is not shown.
      mrkdwn`*Amount*\n${formatPrice(alert.amount)}`,
    ]),
    section(note),
  ];
  if (alert.stripeCustomerId) {
    blocks.push(
      linkButton(
        "View customer in Stripe",
        stripeCustomerUrl(alert.stripeCustomerId, alert.livemode ?? true),
      ),
    );
  }

  await postToSlack(process.env.SLACK_WEBHOOK_CUSTOMERS, `payment_${kind}`, { blocks });
}

export async function alertPaymentFailed(alert: PaymentAlert): Promise<void> {
  await alertPayment(
    alert,
    "failed",
    ":warning:",
    "Payment failed",
    mrkdwn`We should reach out to help resolve the payment issue.`,
  );
}

export async function alertPaymentRecovered(alert: PaymentAlert): Promise<void> {
  await alertPayment(
    alert,
    "recovered",
    ":tada:",
    "Payment recovered",
    mrkdwn`Payment went through after a previous failure. Their service should now be restored.`,
  );
}

/**
 * High-severity ops alert for a live Stripe subscription that could not be
 * linked to (or resolved from) a workspace and cannot be reconciled
 * automatically. Covers a `subscription_conflict` (a race created a second
 * live, charged subscription while the workspace is already served by
 * another), a checkout whose workspace or `client_reference_id` is missing,
 * and a subscription.updated/deleted event whose subscription no workspace row
 * points at. Such a subscription keeps billing until an operator
 * cancels/refunds it, so this must page a human rather than sit in logs.
 *
 * workspaceId and sessionId are optional because the event-driven callers
 * (updated/deleted with no matching workspace, or a checkout missing its
 * client_reference_id) do not have both.
 */
export async function alertOrphanedDeploySubscription(details: {
  workspaceId?: string;
  subscriptionId: string;
  sessionId?: string;
  eventId?: string;
  reason: string;
  product?: "API" | "Compute";
}): Promise<void> {
  const url = process.env.SLACK_WEBHOOK_CUSTOMERS;
  if (!url) {
    return;
  }

  // Include only the fields provided so the message never shows undefined.
  const detailLines = [
    details.workspaceId ? `Workspace: \`${details.workspaceId}\`` : null,
    `Subscription: \`${details.subscriptionId}\``,
    details.sessionId ? `Checkout session: \`${details.sessionId}\`` : null,
    details.eventId ? `Event: \`${details.eventId}\`` : null,
  ].filter((line): line is string => line !== null);

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
              text: `:rotating_light: Orphaned ${details.product ?? "Compute"} subscription needs manual reconciliation (${details.reason})`,
            },
          },
          {
            type: "section",
            text: {
              type: "mrkdwn",
              text: `A live subscription could not be linked to its workspace. The subscription is billing but unlinked — cancel/refund it manually.\n${detailLines.join("\n")}`,
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

/**
 * Ops alert for a subscription.updated whose product carries invalid or
 * missing quota metadata. The handler cannot derive the tier/quota values, so
 * it skips the sync and the workspace silently stays on its old tier while
 * Stripe bills the new plan. Page a human so the product metadata gets fixed.
 */
export async function alertInvalidProductQuotaMetadata(details: {
  productId: string;
  productName: string;
  subscriptionId: string;
  eventId: string;
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
              text: ":rotating_light: Product has invalid quota metadata; tier sync skipped",
            },
          },
          {
            type: "section",
            text: {
              type: "mrkdwn",
              text: `A subscription.updated could not sync the workspace tier because the product's quota metadata is invalid or missing. Fix the product metadata in Stripe.\nProduct: \`${details.productName}\` (\`${details.productId}\`)\nSubscription: \`${details.subscriptionId}\`\nEvent: \`${details.eventId}\``,
            },
          },
        ],
      }),
    });

    if (!response.ok) {
      console.error("Failed to send invalid-quota-metadata alert to Slack:", {
        status: response.status,
        statusText: response.statusText,
        ...details,
      });
    }
  } catch (err: unknown) {
    console.error("Error sending invalid-quota-metadata alert:", {
      error: err instanceof Error ? { message: err.message, name: err.name } : err,
      ...details,
    });
  }
}
