"use client";
import { LogsClient } from "@/app/(app)/[workspace]/apis/[apiId]/_overview/logs-client";
import { useWorkspace } from "@/providers/workspace-provider";
import { ApisNavbar } from "./api-id-navbar";

export default function ApiPage(props: { params: { apiId: string } }) {
  const apiId = props.params.apiId;
  const { workspace } = useWorkspace();

  return (
    <div className="min-h-screen">
      <ApisNavbar
        apiId={apiId}
        activePage={{
          href: `/${workspace?.slug}/apis/${apiId}`,
          text: "Requests",
        }}
      />
      <LogsClient apiId={apiId} />
    </div>
  );
}
