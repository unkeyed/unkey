"use client";;
import { use } from "react";
import { LogsClient } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_overview/logs-client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { ApisNavbar } from "./api-id-navbar";

export default function ApiPage(props: { params: Promise<{ apiId: string }> }) {
  const params = use(props.params);
  const apiId = params.apiId;
  const workspace = useWorkspaceNavigation();

  return (
    <div className="min-h-screen">
      <ApisNavbar
        apiId={apiId}
        activePage={{
          href: `/${workspace.slug}/apis/${apiId}`,
          text: "Requests",
        }}
      />
      <LogsClient apiId={apiId} />
    </div>
  );
}
