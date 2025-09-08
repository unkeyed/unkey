import { useWorkspace } from "@/providers/workspace-provider";

export const navigation = (apiId: string, keyAuthId: string) => {
  const { workspace } = useWorkspace();

  const base = `/${workspace?.slug}/apis/${apiId}`;
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
