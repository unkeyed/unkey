"use client";

import { useWorkspace } from "@/providers/workspace-provider";
import { NamespaceNavbar } from "../namespace-navbar";
import { SettingsClient } from "./components/settings-client";

type Props = {
  params: {
    namespaceId: string;
  };
};

export default function SettingsPage(props: Props) {
  const namespaceId = props.params.namespaceId;
  const { workspace } = useWorkspace();
  return (
    <div>
      <NamespaceNavbar
        namespaceId={namespaceId}
        activePage={{
          href: `/${workspace?.slug}/ratelimits/${namespaceId}/settings`,
          text: "Settings",
        }}
      />
      <SettingsClient namespaceId={namespaceId} />
    </div>
  );
}
