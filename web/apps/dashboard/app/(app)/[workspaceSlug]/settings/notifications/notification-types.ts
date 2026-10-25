export type NotificationCategory = "deployments" | "billing";

export type NotificationTypeDefinition = {
  type: string;
  category: NotificationCategory;
  label: string;
  description: string;
  resourceScoped: boolean;
};

export const NOTIFICATION_CATEGORIES: ReadonlyArray<{
  category: NotificationCategory;
  label: string;
}> = [
  { category: "deployments", label: "Deployments" },
  { category: "billing", label: "Billing" },
];

export const NOTIFICATION_TYPES: ReadonlyArray<NotificationTypeDefinition> = [
  {
    type: "deployment.failed",
    category: "deployments",
    label: "Deployment failed",
    description: "A build or rollout did not finish.",
    resourceScoped: true,
  },
  {
    type: "deployment.rolled_back",
    category: "deployments",
    label: "Deployment rolled back",
    description: "Traffic moved back to an earlier deployment.",
    resourceScoped: true,
  },
  {
    type: "deployment.succeeded",
    category: "deployments",
    label: "Deployment succeeded",
    description: "A deployment is live and serving traffic.",
    resourceScoped: true,
  },
  {
    type: "billing.payment_failed",
    category: "billing",
    label: "Payment failed",
    description: "Unkey could not charge the payment method on file.",
    resourceScoped: false,
  },
  {
    type: "billing.paused",
    category: "billing",
    label: "Workspace paused",
    description: "Deployments stopped because of an unpaid invoice or a spend limit.",
    resourceScoped: false,
  },
];

export function findNotificationType(type: string): NotificationTypeDefinition | undefined {
  return NOTIFICATION_TYPES.find((definition) => definition.type === type);
}
