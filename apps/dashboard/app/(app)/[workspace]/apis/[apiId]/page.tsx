"use client";
import { LogsClient } from "@/app/(app)/[workspace]/apis/[apiId]/_overview/logs-client";
import { useWorkspace } from "@/providers/workspace-provider";
import { redirect, useRouter } from "next/navigation";
import { ApisNavbar } from "./api-id-navbar";

export default function ApiPage(props: { params: { apiId: string } }) {
  const apiId = props.params.apiId;
  const { workspace, error, isLoading } = useWorkspace();
  const router = useRouter();

  if (workspace && !isLoading) {
    router.replace(`/${workspace?.slug}/apis/${apiId}`);
  }

  if (!workspace || error) {
    return redirect("/new");
  }

  return (
    <div className="min-h-screen">
      <ApisNavbar
        apiId={apiId}
        activePage={{
          href: `/${workspace?.slug}/apis/${apiId}`,
          text: "Requests",
        }}
        workspaceSlug={workspace?.slug ?? ""}
      />
      <LogsClient apiId={apiId} workspaceSlug={workspace?.slug ?? ""} />
    </div>
  );
}
