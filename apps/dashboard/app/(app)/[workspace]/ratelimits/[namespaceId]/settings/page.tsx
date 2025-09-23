"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Loading } from "@unkey/ui";
import { Suspense } from "react";
import { NamespaceNavbar } from "../namespace-navbar";
import { SettingsClient } from "./components/settings-client";

type Props = {
  params: {
    namespaceId: string;
  };
};

export default function SettingsPage(props: Props) {
  const namespaceId = props.params.namespaceId;
  const workspace = useWorkspaceNavigation();

  return (
    <div>
      <Suspense fallback={<Loading type="spinner" />}>
        <NamespaceNavbar
          namespaceId={namespaceId}
          activePage={{
            href: `/${workspace.slug}/ratelimits/${namespaceId}/settings`,
            text: "Settings",
          }}
        />
      </Suspense>
      <SettingsClient namespaceId={namespaceId} />
    </div>
  );
}
