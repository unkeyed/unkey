"use client";

import { useWorkspace } from "@/providers/workspace-provider";
import { redirect } from "next/navigation";
import { ApisNavbar } from "../api-id-navbar";
import { SettingsClient } from "./components/settings-client";
type Props = {
  params: {
    apiId: string;
  };
};

export default function SettingsPage(props: Props) {
  const { apiId } = props.params;
  const { workspace, isLoading, error } = useWorkspace();
  if (!workspace && !isLoading && error) {
    return redirect("/new");
  }
  return (
    <div>
      <ApisNavbar
        apiId={apiId}
        activePage={{
          href: `/${workspace?.id}/apis/${apiId}/settings`,
          text: "Settings",
        }}
        workspaceId={workspace?.id ?? ""}
      />
      <SettingsClient apiId={apiId} />
    </div>
  );
}
