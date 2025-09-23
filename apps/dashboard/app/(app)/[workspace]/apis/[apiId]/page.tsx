"use client";
import { LogsClient } from "@/app/(app)/[workspace]/apis/[apiId]/_overview/logs-client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Loading } from "@unkey/ui";
import { Suspense } from "react";
import { ApisNavbar } from "./api-id-navbar";

export default function ApiPage(props: { params: { apiId: string } }) {
  const apiId = props.params.apiId;
  const workspace = useWorkspaceNavigation();

  return (
    <div className="min-h-screen">
      <Suspense fallback={<Loading type="spinner" />}>
        <ApisNavbar
          apiId={apiId}
          activePage={{
            href: `/${workspace.slug}/apis/${apiId}`,
            text: "Requests",
          }}
        />
      </Suspense>
      <LogsClient apiId={apiId} />
    </div>
  );
}
