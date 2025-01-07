export const navigation = (namespaceId: string) => [
  {
    label: "Overview",
    href: `/ratelimits/${namespaceId}`,
    segment: "overview",
  },

  {
    label: "Settings",
    href: `/ratelimits/${namespaceId}/settings`,
    segment: "settings",
  },
  {
    label: "Logs",
    href: `/ratelimits/${namespaceId}/logs`,
    segment: "logs",
  },
  {
    label: "Overrides",
    href: `/ratelimits/${namespaceId}/overrides`,
    segment: "overrides",
  },
];
