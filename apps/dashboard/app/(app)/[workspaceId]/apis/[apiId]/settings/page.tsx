"use client";

import { ApisNavbar } from "../api-id-navbar";
import { SettingsClient } from "./components/settings-client";
type Props = {
  params: {
    apiId: string;
    workspaceId: string;
  };
};

export default function SettingsPage(props: Props) {
  const { apiId, workspaceId } = props.params;
  return (
    <div>
      <ApisNavbar
        apiId={apiId}
        activePage={{
          href: `/${workspaceId}/apis/${apiId}/settings`,
          text: "Settings",
        }}
        workspaceId={workspaceId}
      />
      <SettingsClient apiId={apiId} />
    </div>
  );
}
