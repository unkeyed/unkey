"use client";

import { Navbar } from "@/components/navigation/navbar";
import { Nodes } from "@unkey/icons";
import { CreateApiButton } from "./_components/create-api-button";

type NavigationProps = {
  isNewApi: boolean;
  apisLength: number;
};

export function Navigation({ isNewApi, apisLength }: NavigationProps) {
  return (
    <Navbar>
      <Navbar.Breadcrumbs icon={<Nodes />}>
        <Navbar.Breadcrumbs.Link href="/apis" active>
          APIs
        </Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <Navbar.Actions>
        <CreateApiButton key="createApi" defaultOpen={apisLength === 0 || isNewApi} />
      </Navbar.Actions>
    </Navbar>
  );
}
