"use client";

import { Navbar } from "@/components/navigation/navbar";
import { useWorkspace } from "@/providers/workspace-provider";
import { Nodes } from "@unkey/icons";
import { useRouter, useSearchParams } from "next/navigation";
import { ApiListClient } from "./_components/api-list-client";
import { CreateApiButton } from "./_components/create-api-button";

export default function ApisOverviewPage() {
  const { workspace, isLoading } = useWorkspace();
  const router = useRouter();

  if (workspace && !isLoading) {
    router.replace(`/${workspace.id}/apis`);
  }

  const searchParams = useSearchParams();
  const isNewApi = searchParams?.get("new") === "true";

  if (!workspace && !isLoading) {
    router.push("/new");
  }

  return (
    <div>
      <Navbar>
        <Navbar.Breadcrumbs icon={<Nodes />}>
          <Navbar.Breadcrumbs.Link href={`/${workspace?.id}/apis`} active>
            APIs
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <CreateApiButton
            key="createApi"
            defaultOpen={isNewApi}
            workspaceId={workspace?.id ?? ""}
          />
        </Navbar.Actions>
      </Navbar>
      <ApiListClient workspaceId={workspace?.id ?? ""} />
    </div>
  );
}
