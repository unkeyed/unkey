export async function alertSubscriptionCreation(
  product: string,
  price: string,
  email: string,
  name?: string,
): Promise<void> {
  const url = process.env.SLACK_WEBHOOK_CUSTOMERS;
  if (!url) {
    return;
  }

  await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      blocks: [
        {
          type: "section",
          text: {
            type: "mrkdwn",
            text: `:bugeyes: New customer ${name || "Unknown"} signed up`,
          },
        },
        {
          type: "section",
          text: {
            type: "mrkdwn",
            text: `A new subscription for the ${product} tier has started at a price of ${price} by ${email} :moneybag: `,
          },
        },
      ],
    }),
  }).catch((err: Error) => {
    console.error(err);
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
  const url = process.env.SLACK_WEBHOOK_CUSTOMERS;
  if (!url) {
    return;
  }

  // Choose appropriate emoji based on change type
  let emoji = ":stonks:";
  let actionText = "updated their subscription";

  if (changeType === "upgraded") {
    actionText = "upgraded their subscription";
  } else if (changeType === "downgraded") {
    emoji = ":notstonks:";
    actionText = "downgraded their subscription";
  }

  // Build the subscription change message
  let subscriptionText = `Subscription ${changeType} to the ${product} tier`;
  if (previousTier && changeType !== "updated") {
    subscriptionText = `${
      name || "Unknown"
    }'s subscription ${changeType} from ${previousTier} to ${product} tier, they are now paying ${price}. `;
  }

  const contactInfo = `Here is their contact information: ${email}`;

  await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      blocks: [
        {
          type: "section",
          text: {
            type: "mrkdwn",
            text: `${emoji} ${name || "Unknown"} ${actionText}`,
          },
        },
        {
          type: "section",
          text: {
            type: "mrkdwn",
            text: subscriptionText,
          },
        },
        {
          type: "section",
          text: {
            type: "mrkdwn",
            text: contactInfo,
          },
        },
      ],
    }),
  }).catch((err: Error) => {
    console.error(err);
  });
}

export async function alertIsCancellingSubscription(
  product: string,
  price: string,
  email: string,
  name?: string,
): Promise<void> {
  const url = process.env.SLACK_WEBHOOK_CUSTOMERS;
  if (!url) {
    return;
  }

  await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      blocks: [
        {
          type: "section",
          text: {
            type: "mrkdwn",
            text: `:warning: ${name || "Unknown"} is cancelling their subscription.`,
          },
        },
        {
          type: "section",
          text: {
            type: "mrkdwn",
            text: `Subscription cancellation requested by ${email} - for ${product} at ${price} they will be moved back to the free tier, at the end of the month. We should reach out to find out why they are cancelling.`,
          },
        },
      ],
    }),
  }).catch((err: Error) => {
    console.error(err);
  });
}

export async function alertPaymentFailed(
  product: string,
  price: string,
  email: string,
  name?: string,
): Promise<void> {
  const url = process.env.SLACK_WEBHOOK_CUSTOMERS;
  if (!url) {
    return;
  }

  await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      blocks: [
        {
          type: "section",
          text: {
            type: "mrkdwn",
            text: `:warning: Payment failed for ${name || "Unknown"}`,
          },
        },
        {
          type: "section",
          text: {
            type: "mrkdwn",
            text: `Payment for ${product} subscription at ${price} failed for ${email}. Subscription is now past due. Please reach out to customer to update payment method.`,
          },
        },
      ],
    }),
  }).catch((err: Error) => {
    console.error(err);
  });
}

export async function alertSubscriptionCancelled(email: string, name?: string): Promise<void> {
  const url = process.env.SLACK_WEBHOOK_CUSTOMERS;
  if (!url) {
    return;
  }

  await fetch(url, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      blocks: [
        {
          type: "section",
          text: {
            type: "mrkdwn",
            text: `:caleb-sad: ${name || "Unknown"} cancelled their subscription`,
          },
        },
        {
          type: "section",
          text: {
            type: "mrkdwn",
            text: `Subscription cancelled by ${email} - they've been moved back to the free tier`,
          },
        },
      ],
    }),
  }).catch((err: Error) => {
    console.error(err);
  });
}
