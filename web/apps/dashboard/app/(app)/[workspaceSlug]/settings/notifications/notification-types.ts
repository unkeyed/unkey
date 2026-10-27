export type NotificationCategory = "deployments";

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
}> = [{ category: "deployments", label: "Deployments" }];

export const NOTIFICATION_TYPES: ReadonlyArray<NotificationTypeDefinition> = [
  {
    type: "deployment.failed",
    category: "deployments",
    label: "Deployment failed",
    description: "A build or rollout did not finish.",
    resourceScoped: true,
  },
  {
    type: "deployment.preparing",
    category: "deployments",
    label: "Deployment preparing",
    description: "A deployment is preparing its build or image.",
    resourceScoped: true,
  },
  {
    type: "deployment.succeeded",
    category: "deployments",
    label: "Deployment succeeded",
    description: "A deployment completed, with or without changing live traffic.",
    resourceScoped: true,
  },
  {
    type: "deployment.started",
    category: "deployments",
    label: "Deployment started",
    description: "A deployment is starting in its regions.",
    resourceScoped: true,
  },
  {
    type: "deployment.routing",
    category: "deployments",
    label: "Deployment routing",
    description: "Routing is being configured for a deployment.",
    resourceScoped: true,
  },
  {
    type: "deployment.cancelled",
    category: "deployments",
    label: "Deployment cancelled",
    description: "A deployment stopped before completion.",
    resourceScoped: true,
  },
  {
    type: "deployment.superseded",
    category: "deployments",
    label: "Deployment superseded",
    description: "A newer deployment replaced this deployment.",
    resourceScoped: true,
  },
];

export function supportedNotificationTypes(
  eventTypes: ReadonlyArray<string>,
): NotificationTypeDefinition[] {
  return NOTIFICATION_TYPES.filter((definition) => eventTypes.includes(definition.type));
}

export function findNotificationType(type: string): NotificationTypeDefinition | undefined {
  return NOTIFICATION_TYPES.find((definition) => definition.type === type);
}
