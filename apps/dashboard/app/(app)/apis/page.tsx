"use client";

import { Navbar } from "@/components/navigation/navbar";
import { Nodes } from "@unkey/icons";
import { useSearchParams } from "next/navigation";
import { ApiListWrapper } from "./_components/api-list-wrapper";
import { CreateApiButton } from "./_components/create-api-button";

export default function ApisOverviewPage() {
  const searchParams = useSearchParams();
  const isNewApi = searchParams?.get("new") === "true";

  return (
    <div>
      <Navbar>
        <Navbar.Breadcrumbs icon={<Nodes />}>
          <Navbar.Breadcrumbs.Link href="/apis" active>
            APIs
          </Navbar.Breadcrumbs.Link>
        </Navbar.Breadcrumbs>
        <Navbar.Actions>
          <CreateApiButton key="createApi" defaultOpen={isNewApi} />
        </Navbar.Actions>
      </Navbar>
      <ApiListWrapper />
    </div>
  );
}
