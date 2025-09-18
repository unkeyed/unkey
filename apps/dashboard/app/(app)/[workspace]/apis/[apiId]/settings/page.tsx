"use client";

import { useWorkspace } from "@/providers/workspace-provider";
import { ApisNavbar } from "../api-id-navbar";
import { SettingsClient } from "./components/settings-client";
type Props = {
  params: {
    apiId: string;
  };
};

export default function SettingsPage(props: Props) {
  const { apiId } = props.params;
  const { workspace } = useWorkspace();

  return (
    <div>
      <ApisNavbar
        apiId={apiId}
        activePage={{
          href: `/${workspace?.slug}/apis/${apiId}/settings`,
          text: "Settings",
        }}
      />
      <SettingsClient apiId={apiId} />
    </div>
  );
}
