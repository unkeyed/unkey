"use client";
import { LogsClient } from "@/app/(app)/[workspaceId]/apis/[apiId]/_overview/logs-client";
import { useSearchParams } from "next/navigation";
import { ApisNavbar } from "./api-id-navbar";

export default function ApiPage(props: { params: { apiId: string; workspaceId: string } }) {
  const apiId = props.params.apiId;
  const searchParams = useSearchParams();
  const workspaceId = props.params.workspaceId ?? searchParams?.get("workspace");

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
