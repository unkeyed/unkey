export const navigation = (apiId: string, keyAuthId: string, workspaceId: string) => [
  {
    label: "Overview",
    href: `/${workspaceId}/apis/${apiId}`,
    segment: "overview",
  },
  {
    label: "Keys",
    href: `/${workspaceId}/apis/${apiId}/keys/${keyAuthId}`,
    segment: "keys",
  },
  {
    label: "API Settings",
    href: `/${workspaceId}/apis/${apiId}/settings`,
    segment: "settings",
  },
];
