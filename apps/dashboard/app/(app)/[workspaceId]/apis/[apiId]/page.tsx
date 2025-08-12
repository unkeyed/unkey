"use client";
import { LogsClient } from "@/app/(app)/[workspaceId]/apis/[apiId]/_overview/logs-client";
import { useWorkspace } from "@/providers/workspace-provider";
import { redirect } from "next/navigation";
import { ApisNavbar } from "./api-id-navbar";

export default function ApiPage(props: { params: { apiId: string } }) {
  const apiId = props.params.apiId;
  const { workspace, isLoading, error } = useWorkspace();
  if (!workspace && !isLoading && error) {
    return redirect("/new");
  }
  return (
    <div className="min-h-screen">
      <ApisNavbar
        apiId={apiId}
        activePage={{
          href: `/${workspace?.id}/apis/${apiId}`,
          text: "Requests",
        }}
        workspaceId={workspace?.id ?? ""}
      />
      <LogsClient apiId={apiId} workspaceId={workspace?.id ?? ""} />
    </div>
  );
}
