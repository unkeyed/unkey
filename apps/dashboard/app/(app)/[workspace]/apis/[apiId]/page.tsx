"use client";
import { LogsClient } from "@/app/(app)/[workspace]/apis/[apiId]/_overview/logs-client";
import { ApisNavbar } from "./api-id-navbar";

export default function ApiPage(props: { params: { apiId: string; workspaceId: string } }) {
  const apiId = props.params.apiId;
  const workspaceId = props.params.workspaceId;

  return (
    <div className="min-h-screen">
      <ApisNavbar
        apiId={apiId}
        activePage={{
          href: `/${workspaceId}/apis/${apiId}`,
          text: "Requests",
        }}
        workspaceId={workspaceId}
      />
      <LogsClient apiId={apiId} workspaceId={workspaceId} />
    </div>
  );
}
