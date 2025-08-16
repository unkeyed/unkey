"use client";

import { NamespaceNavbar } from "../namespace-navbar";
import { SettingsClient } from "./components/settings-client";

type Props = {
  params: {
    workspaceId: string;
    namespaceId: string;
  };
};

export default function SettingsPage(props: Props) {
  const { workspaceId } = props.params;
  const namespaceId = props.params.namespaceId;
  return (
    <div>
      <NamespaceNavbar
        namespaceId={namespaceId}
        activePage={{
          href: `/${workspaceId}/ratelimits/${namespaceId}/settings`,
          text: "Settings",
        }}
        workspaceId={workspaceId}
      />
      <SettingsClient namespaceId={namespaceId} />
    </div>
  );
}
