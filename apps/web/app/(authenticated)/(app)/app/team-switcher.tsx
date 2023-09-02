"use client";

import { Loading } from "@/components/dashboard/loading";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Check, ChevronsUpDown, Plus } from "lucide-react";
import { useRouter } from "next/navigation";
import React, { useMemo, useState } from "react";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { useOrganization, useOrganizationList, useUser } from "@clerk/nextjs";

import Link from "next/link";

type Props = {};

export const WorkspaceSwitcher: React.FC<Props> = (): JSX.Element => {
  const { setActive, organizationList, isLoaded: clerkLoaded } = useOrganizationList();
  const { organization: currentOrg, membership } = useOrganization();
  const { user } = useUser();
  const _router = useRouter();
  const [isLoading, setLoading] = useState(false);

  async function changeOrg(orgId: string | null) {
    if (!setActive) {
      return;
    }
    try {
      setLoading(true);
      await setActive({
        organization: orgId,
      });
    } finally {
      setLoading(false);
    }
  }
  const [search, _setSearch] = useState("");
  // rome-ignore lint: suspicious/noNonNullAssertion
  const filteredOrgs = useMemo(() => {
    if (!organizationList) {
      return [];
    }
    if (search === "") {
      return organizationList;
    }
    console.log({ search });
    return organizationList?.filter(
      ({ organization }) =>
        organization.name.toLowerCase().includes(search.toLowerCase()) ||
        organization.slug?.toLowerCase().includes(search.toLowerCase()),
    );
  }, [search, organizationList])!;
  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="flex items-center justify-between w-full gap-2">
        <div className="flex items-center gap-2">
          <Avatar className="w-6 h-6">
            {currentOrg?.imageUrl ? (
              <AvatarImage src={currentOrg.imageUrl} alt={currentOrg.name ?? "Profile picture"} />
            ) : user?.imageUrl ? (
              <AvatarImage src={user.imageUrl} alt={user.username ?? "Profile picture"} />
            ) : null}
            <AvatarFallback className="flex items-center justify-center w-8 h-8 overflow-hidden text-gray-700 bg-gray-100 border border-gray-500 rounded">
              {(currentOrg?.name ?? user?.fullName ?? "").slice(0, 2).toUpperCase() ?? "P"}
            </AvatarFallback>
          </Avatar>
          {!clerkLoaded || isLoading ? (
            <Loading />
          ) : (
            <span className="text-sm font-semibold">
              {currentOrg?.name ?? "Personal Workspace"}
            </span>
          )}
        </div>
        <ChevronsUpDown className="hidden w-3 h-3 md:block" />
      </DropdownMenuTrigger>
      <DropdownMenuContent side="right" className="w-96">
        <DropdownMenuLabel>Personal Account</DropdownMenuLabel>
        <DropdownMenuItem
          className="flex items-center justify-between"
          onClick={() => changeOrg(null)}
        >
          <span className={currentOrg === null ? "font-semibold" : undefined}>
            {user?.fullName}
          </span>
          {currentOrg === null ? <Check className="w-4 h-4" /> : null}
        </DropdownMenuItem>
        <DropdownMenuSeparator />

        <DropdownMenuLabel>Workspaces</DropdownMenuLabel>
        <DropdownMenuGroup>
          {filteredOrgs.map(({ organization }) => (
            <DropdownMenuItem
              key={organization.id}
              className="flex items-center justify-between"
              onClick={() => changeOrg(organization.id)}
            >
              <span className={organization.id === currentOrg?.id ? "font-semibold" : undefined}>
                {" "}
                {organization.name}
              </span>
              {organization.id === currentOrg?.id ? <Check className="w-4 h-4" /> : null}
            </DropdownMenuItem>
          ))}
          <DropdownMenuSeparator />

          <DropdownMenuItem>
            <Link href="/new" className="flex items-center">
              <Plus className="w-4 h-4 mr-2" />
              <span>Create Workspace</span>
            </Link>
            {/* <DropdownMenuShortcut>⌘B</DropdownMenuShortcut> */}
          </DropdownMenuItem>
          {membership?.role === "admin" ? (
            <Link href="/app/team/invite">
              <DropdownMenuItem>
                <Plus className="w-4 h-4 mr-2 " />
                <span className="cursor-pointer">Invite Member</span>
              </DropdownMenuItem>
            </Link>
          ) : null}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
