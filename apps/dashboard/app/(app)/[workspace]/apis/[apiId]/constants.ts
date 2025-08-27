export const navigation = (apiId: string, keyAuthId: string, workspaceId: string) => {
  const base = `/${workspaceId}/apis/${apiId}`;
  return [
    {
      label: "Requests",
      href: base,
      segment: "requests",
    },
    {
      label: "Keys",
      href: `${base}/keys/${keyAuthId}`,
      segment: "keys",
    },
    {
      label: "Settings",
      href: `${base}/settings`,
      segment: "settings",
    },
  ];
};
