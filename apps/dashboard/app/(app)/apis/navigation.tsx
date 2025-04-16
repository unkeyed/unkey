"use client";

import { Navbar } from "@/components/navigation/navbar";
import { trpc } from "@/lib/trpc/client";
import { Nodes } from "@unkey/icons";
import { CreateApiButton } from "./_components/create-api-button";

type NavigationProps = {
  isNewApi: boolean;
};

export function Navigation({ isNewApi }: NavigationProps) {
  const { data, isLoading } = trpc.api.overview.query.useQuery({
    cursor: undefined,
  });
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Nodes />}>
        <Navbar.Breadcrumbs.Link href="/apis" active>
          APIs
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <Navbar.Actions>
        <CreateApiButton
          key="createApi"
          defaultOpen={(!isLoading && data?.total === 0) || isNewApi}
        />
      </Navbar.Actions>
    </Navbar>
  );
}
