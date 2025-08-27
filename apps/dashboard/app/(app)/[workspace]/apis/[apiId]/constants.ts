export const navigation = (apiId: string, keyAuthId: string, workspaceSlug: string) => {
  const base = `/${workspaceSlug}/apis/${apiId}`;
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
