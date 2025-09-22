"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { ApisNavbar } from "../api-id-navbar";
import { SettingsClient } from "./components/settings-client";
import { Loading } from "@unkey/ui";
import { redirect } from "next/navigation";
type Props = {
  params: {
    apiId: string;
  };
};

export default function SettingsPage(props: Props) {
  const { apiId } = props.params;
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
