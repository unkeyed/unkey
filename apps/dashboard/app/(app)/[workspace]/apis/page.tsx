"use client";

import { Navbar } from "@/components/navigation/navbar";
import { useWorkspace } from "@/providers/workspace-provider";
import { Nodes } from "@unkey/icons";
import { Loading } from "@unkey/ui";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect } from "react";
import { ApiListClient } from "./_components/api-list-client";
import { CreateApiButton } from "./_components/create-api-button";

export default function ApisOverviewPage() {
  const { workspace, isLoading } = useWorkspace();
  const router = useRouter();

  const searchParams = useSearchParams();
  const isNewApi = searchParams?.get("new") === "true";

  if (!workspace && !isLoading) {
    router.push("/new");
  }

  return (
    workspace && (
      <div>
        <Navbar>
          <Navbar.Breadcrumbs icon={<Nodes />}>
            <Navbar.Breadcrumbs.Link href={`/${workspace?.slug}/apis`} active>
              APIs
            </Navbar.Breadcrumbs.Link>
          </Navbar.Breadcrumbs>
          <Navbar.Actions>
            <CreateApiButton
              key="createApi"
              defaultOpen={isNewApi}
              workspaceSlug={workspace.slug ?? ""}
            />
          </Navbar.Actions>
        </Navbar>
        <ApiListClient workspaceSlug={workspace.slug ?? ""} />
      </div>
    )
  );
}
