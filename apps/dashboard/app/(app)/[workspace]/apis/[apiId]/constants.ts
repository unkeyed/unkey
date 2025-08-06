export const navigation = (apiId: string, keyAuthId: string) => [
  {
    label: "Overview",
    href: `/apis/${apiId}`,
    segment: "overview",
  },
  {
    label: "Keys",
    href: `/apis/${apiId}/keys/${keyAuthId}`,
    segment: "keys",
  },
  {
    label: "API Settings",
    href: `/apis/${apiId}/settings`,
    segment: "settings",
  },
];
