"use client";

import { useWorkspace } from "@/providers/workspace-provider";
import { redirect, useRouter } from "next/navigation";
import { ApisNavbar } from "../api-id-navbar";
import { SettingsClient } from "./components/settings-client";
type Props = {
  params: {
    apiId: string;
  };
};

export default function SettingsPage(props: Props) {
  const { apiId } = props.params;
  const { workspace, error } = useWorkspace();
  const router = useRouter();

  if (!workspace || error) {
    return redirect("/new");
  }

  router.replace(`/${workspace?.slug}/apis/${apiId}/settings`);

  return (
    <div>
      <ApisNavbar
        apiId={apiId}
        activePage={{
          href: `/${workspace?.slug}/apis/${apiId}/settings`,
          text: "Settings",
        }}
        workspaceSlug={workspace?.slug ?? ""}
      />
      <SettingsClient apiId={apiId} workspaceSlug={workspace?.slug ?? ""} />
    </div>
  );
}
