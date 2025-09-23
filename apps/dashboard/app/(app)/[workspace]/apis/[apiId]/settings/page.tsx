"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Loading } from "@unkey/ui";
import { Suspense } from "react";
import { ApisNavbar } from "../api-id-navbar";
import { SettingsClient } from "./components/settings-client";

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
      <Suspense fallback={<Loading type="spinner" />}>
        <ApisNavbar
          apiId={apiId}
          activePage={{
            href: `/${workspace.slug}/apis/${apiId}/settings`,
            text: "Settings",
          }}
        />
        <SettingsClient apiId={apiId} />
      </Suspense>
    </div>
  );
}
