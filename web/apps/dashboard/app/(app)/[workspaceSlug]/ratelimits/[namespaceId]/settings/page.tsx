"use client";
import { use } from "react";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { NamespaceNavbar } from "../namespace-navbar";
import { SettingsClient } from "./components/settings-client";

type Props = {
  params: Promise<{
    namespaceId: string;
  }>;
};

export default function SettingsPage(props: Props) {
  const params = use(props.params);
  const namespaceId = params.namespaceId;
  const workspace = useWorkspaceNavigation();

  return (
    <div>
      <NamespaceNavbar
        namespaceId={namespaceId}
        activePage={{
          href: routes.ratelimits.settings({ workspaceSlug: workspace.slug, namespaceId }),
          text: "Settings",
        }}
      />
      <SettingsClient namespaceId={namespaceId} />
    </div>
  );
}
