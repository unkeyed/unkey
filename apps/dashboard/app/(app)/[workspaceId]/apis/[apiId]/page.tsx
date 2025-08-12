"use client";
import { LogsClient } from "@/app/(app)/[workspaceId]/apis/[apiId]/_overview/logs-client";
import { useWorkspace } from "@/providers/workspace-provider";
import { Loading } from "@unkey/ui";
import { redirect, useRouter } from "next/navigation";
import { ApisNavbar } from "./api-id-navbar";

export default function ApiPage(props: { params: { apiId: string } }) {
  const apiId = props.params.apiId;
  const { workspace, isLoading, error } = useWorkspace();
  const router = useRouter();

  if (isLoading) {
    return <Loading size={18} />;
  }

  if ((!workspace && !isLoading) || error) {
    return redirect("/new");
  }

  router.replace(`/${workspace?.id}/apis/${apiId}`);

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
