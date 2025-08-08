"use client";

import { Navbar } from "@/components/navigation/navbar";
import { Nodes } from "@unkey/icons";
import { useSearchParams } from "next/navigation";
import { ApiListClient } from "./_components/api-list-client";
import { CreateApiButton } from "./_components/create-api-button";

export default function ApisOverviewPage({ params }: { params: { workspaceId: string } }) {
  const searchParams = useSearchParams();
  const isNewApi = searchParams?.get("new") === "true";
  const workspaceId = searchParams?.get("workspace") ?? params.workspaceId;

  return (
    <div>
      <Navbar>
        <Navbar.Breadcrumbs icon={<Nodes />}>
          <Navbar.Breadcrumbs.Link href={`/${workspaceId}/apis`} active>
            APIs
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <CreateApiButton key="createApi" defaultOpen={isNewApi} workspaceId={workspaceId} />
        </Navbar.Actions>
      </Navbar>
      <ApiListClient workspaceId={workspaceId} />
    </div>
  );
}
