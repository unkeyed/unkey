"use client";
import { use } from "react";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { ApisNavbar } from "../api-id-navbar";
import { SettingsClient } from "./components/settings-client";

type Props = {
  params: Promise<{
    apiId: string;
  }>;
};

export default function SettingsPage(props: Props) {
  const params = use(props.params);
  const { apiId } = params;
  const workspace = useWorkspaceNavigation();

  return (
    <div>
      <ApisNavbar
        apiId={apiId}
        activePage={{
          href: `/${workspace.slug}/apis/${apiId}/settings`,
          text: "Settings",
        }}
      />
      <SettingsClient apiId={apiId} />
    </div>
  );
}
