"use client";
import { LogsClient } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_overview/logs-client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { use } from "react";
import { ApisNavbar } from "./api-id-navbar";

export default function ApiPage(props: { params: Promise<{ apiId: string }> }) {
  const params = use(props.params);
  const apiId = params.apiId;
  const workspace = useWorkspaceNavigation();

  return (
    <div>
      <ApisNavbar
        apiId={apiId}
        activePage={{
          href: routes.apis.detail({ workspaceSlug: workspace.slug, apiId }),
          text: "Requests",
        }}
      />
      <LogsClient apiId={apiId} />
    </div>
  );
}
