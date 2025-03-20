"use client";
import { Navbar } from "@/components/navigation/navbar";
import { CreateApiButton } from "./_components/create-api-button";

type NavigationProps = {
  isNewApi: boolean;
  apisLength: number;
};

export function Navigation({ isNewApi, apisLength }: NavigationProps) {
  return (
    <Navbar>
      <Navbar.Breadcrumbs>
        <Navbar.Breadcrumbs.Link href="/apis">APIs</Navbar.Breadcrumbs.Link>
      </Navbar.Breadcrumbs>
      <Navbar.Actions>
        <CreateApiButton key="createApi" defaultOpen={apisLength === 0 || isNewApi} />
      </Navbar.Actions>
    </Navbar>
  );
}
