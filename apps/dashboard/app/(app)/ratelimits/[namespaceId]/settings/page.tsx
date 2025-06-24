"use client";

import { NamespaceNavbar } from "../namespace-navbar";
import { SettingsClient } from "./components/settings-client";

type Props = {
  params: {
    namespaceId: string;
  };
};

export default function SettingsPage(props: Props) {
  const namespaceId = props.params.namespaceId;
  return (
    <div>
      <NamespaceNavbar
        namespaceId={namespaceId}
        activePage={{
          href: `/ratelimits/${namespaceId}/settings`,
          text: "Settings",
        }}
      />
      <SettingsClient namespaceId={namespaceId} />
    </div>
  );
}
