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

type ContextBlock = { type: "context"; elements: MrkdwnText[] };

type SlackBlock = HeaderBlock | SectionBlock | ContextBlock;

const mrkdwnText = (text: Mrkdwn): MrkdwnText => {
  return { type: "mrkdwn", text, verbatim: true };
};

const plainText = (text: string): PlainText => {
  return { type: "plain_text", text, emoji: true };
};

type SlackMessage = {
  text?: Mrkdwn;
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

const context = (text: Mrkdwn): ContextBlock => {
  return { type: "context", elements: [mrkdwnText(text)] };
};

function stripeCustomerBlocks(customerId: string | undefined, livemode = true): SlackBlock[] {
  if (!customerId) {
    return [];
  }
  return [
    section(mrkdwn`<${stripeCustomerUrl(customerId, livemode)}|View customer in Stripe>`),
    context(mrkdwn`Stripe customer: \`${customerId}\``),
  ];
}

/**
 * A deep link to the customer in the Stripe dashboard. Test-mode objects live under `/test`, so
 * the link must switch on `livemode` or it 404s in one of the two modes. The id is Stripe-issued
 * (`cus_...`), not customer free text, but it is still encoded before going into the URL.
 */
export function stripeCustomerUrl(customerId: string, livemode: boolean): string {
  const base = livemode ? "https://dashboard.stripe.com" : "https://dashboard.stripe.com/test";
  return `${base}/customers/${encodeURIComponent(customerId)}`;
}

export type CustomerAlertAction =
  | "signup"
  | "upgrade"
  | "downgrade"
  | "update"
  | "cancelling"
  | "cancelled";

const ACTION_META: Record<CustomerAlertAction, { title: string; note?: Mrkdwn }> = {
  signup: { title: "New customer signup" },
  upgrade: { title: "Subscription upgraded" },
  downgrade: { title: "Subscription downgraded" },
  update: { title: "Subscription updated" },
  cancelling: {
    title: "Subscription cancelling",
    note: mrkdwn`They stay on their plan until the end of the billing period, then move back to the free tier. Worth reaching out to learn why.`,
  },
  cancelled: {
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

export async function alertCustomerLifecycle(
  alert: CustomerLifecycleAlert,
): Promise<SlackPostStatus> {
  const meta = ACTION_META[alert.action];
  const plan = alert.product
    ? alert.previousProduct
      ? `${alert.previousProduct} → ${alert.product}`
      : alert.product
    : undefined;
  const planDetails = [plan, alert.price].filter(Boolean).join(" · ");

  const blocks: SlackBlock[] = [header(meta.title)];
  if (meta.note) {
    blocks.push(section(meta.note));
  }
  const fields: Mrkdwn[] = [];
  if (alert.workspaceName) {
    fields.push(mrkdwn`*Workspace*\n${alert.workspaceName}`);
  }
  if (planDetails) {
    fields.push(plan ? mrkdwn`*Plan*\n${planDetails}` : mrkdwn`*Price*\n${planDetails}`);
  }
  if (fields.length > 0) {
    blocks.push(fieldsSection(fields));
  }
  blocks.push(section(mrkdwn`*Customer*\n${alert.name}\n${alert.email}`));
  blocks.push(...stripeCustomerBlocks(alert.stripeCustomerId, alert.livemode));
  if (alert.workspaceId) {
    blocks.push(context(mrkdwn`Workspace ID: \`${alert.workspaceId}\``));
  }
  const summary = [
    `${meta.title}.`,
    `Customer: ${alert.name} (${alert.email}).`,
    alert.workspaceName ? `Workspace: ${alert.workspaceName}.` : undefined,
    planDetails ? `${plan ? "Plan" : "Price"}: ${planDetails}.` : undefined,
    meta.note,
  ]
    .filter(Boolean)
    .join(" ");
  return postToSlack(process.env.SLACK_WEBHOOK_CUSTOMERS, `subscription_${alert.action}`, {
    text: mrkdwn`${summary}`,
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

async function alertPayment(
  alert: PaymentAlert,
  kind: "failed" | "recovered",
  title: string,
  note: Mrkdwn,
): Promise<void> {
  const amount = formatPrice(alert.amount);
  const blocks: SlackBlock[] = [
    header(title),
    section(note),
    section(mrkdwn`*Amount*\n${amount}`),
    section(mrkdwn`*Customer*\n${alert.name}\n${alert.email}`),
    ...stripeCustomerBlocks(alert.stripeCustomerId, alert.livemode),
  ];

  await postToSlack(process.env.SLACK_WEBHOOK_CUSTOMERS, `payment_${kind}`, {
    text: mrkdwn`${title}: ${amount}. Customer: ${alert.name} (${alert.email}). ${note}`,
    blocks,
  });
}

export async function alertPaymentFailed(alert: PaymentAlert): Promise<void> {
  await alertPayment(
    alert,
    "failed",
    "Payment failed",
    mrkdwn`We should reach out to help resolve the payment issue.`,
  );
}

export async function alertPaymentRecovered(alert: PaymentAlert): Promise<void> {
  await alertPayment(
    alert,
    "recovered",
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
  livemode?: boolean;
}): Promise<void> {
  const product = details.product ?? "Compute";
  const reason =
    details.reason === "subscription_conflict" ? "Subscription conflict" : details.reason;
  const fields = [mrkdwn`*Product*\n${product}`, mrkdwn`*Reason*\n${reason}`];
  if (details.workspaceId) {
    fields.push(mrkdwn`*Workspace ID*\n\`${details.workspaceId}\``);
  }
  fields.push(mrkdwn`*Subscription*\n\`${details.subscriptionId}\``);
  const subscriptionUrl = stripeResourceUrl(
    "subscriptions",
    details.subscriptionId,
    details.livemode,
  );
  const blocks: SlackBlock[] = [
    header(`${product} subscription needs attention`),
    section(
      mrkdwn`This subscription is still billing but is not linked to its workspace.\nReview it in Stripe and cancel or refund it manually.`,
    ),
    fieldsSection(fields),
    section(mrkdwn`<${subscriptionUrl}|Review subscription in Stripe>`),
  ];
  if (details.sessionId) {
    blocks.push({
      type: "context",
      elements: [mrkdwnText(mrkdwn`Checkout session: \`${details.sessionId}\``)],
    });
  }
  if (details.eventId) {
    blocks.push({
      type: "context",
      elements: [mrkdwnText(mrkdwn`Stripe event: \`${details.eventId}\``)],
    });
  }
  await postToSlack(process.env.SLACK_WEBHOOK_CUSTOMERS, "orphaned_subscription", {
    text: mrkdwn`${product} subscription ${details.subscriptionId} is billing but unlinked. Review it in Stripe and cancel or refund it manually.`,
    blocks,
  });
}

function stripeResourceUrl(
  resource: "subscriptions" | "products",
  id: string,
  livemode = true,
): string {
  const base = livemode ? "https://dashboard.stripe.com" : "https://dashboard.stripe.com/test";
  return `${base}/${resource}/${encodeURIComponent(id)}`;
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
  livemode?: boolean;
}): Promise<void> {
  const productUrl = stripeResourceUrl("products", details.productId, details.livemode);
  const subscriptionUrl = stripeResourceUrl(
    "subscriptions",
    details.subscriptionId,
    details.livemode,
  );
  await postToSlack(process.env.SLACK_WEBHOOK_CUSTOMERS, "invalid_product_quota_metadata", {
    text: mrkdwn`${details.productName}: workspace tier sync was skipped because quota metadata is invalid or missing. Fix the product metadata in Stripe.`,
    blocks: [
      header("Product quota settings need attention"),
      section(
        mrkdwn`Workspace tier sync was skipped.\nFix the invalid or missing product quota metadata in Stripe.`,
      ),
      fieldsSection([
        mrkdwn`*Product*\n${details.productName}`,
        mrkdwn`*Subscription*\n\`${details.subscriptionId}\``,
      ]),
      section(
        mrkdwn`<${productUrl}|Review product in Stripe> · <${subscriptionUrl}|View subscription>`,
      ),
      { type: "context", elements: [mrkdwnText(mrkdwn`Product ID: \`${details.productId}\``)] },
      { type: "context", elements: [mrkdwnText(mrkdwn`Stripe event: \`${details.eventId}\``)] },
    ],
  });
}
