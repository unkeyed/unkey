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
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Check, ChevronsUpDown, Plus, UserPlus } from "lucide-react";
import { useRouter } from "next/navigation";
import React, { useMemo, useState } from "react";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { useOrganization, useOrganizationList, useUser } from "@clerk/nextjs";

import Link from "next/link";

export const WorkspaceSwitcher: React.FC = (): JSX.Element => {
  const { isLoaded, setActive, userMemberships } = useOrganizationList({
    userMemberships: {
      infinite: true,
      pageSize: 100,
    },
  });
  const { organization: currentOrg, membership } = useOrganization();
  const { user } = useUser();
  const router = useRouter();
  async function changeOrg(orgId: string | null) {
    if (!setActive) {
      return;
    }
    try {
      await setActive({
        organization: orgId,
      });
    } finally {
      router.refresh();
    }
  }
  const [search, _setSearch] = useState("");
  const filteredOrgs = useMemo(() => {
    if (!userMemberships.data) {
      return [];
    }
    if (search === "") {
      return userMemberships.data;
    }
    return userMemberships.data?.filter(({ organization }) =>
      organization.name.toLowerCase().includes(search.toLowerCase()),
    );
  }, [search, userMemberships])!;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="h-12 flex items-center px-2 hover:bg-gray-200 dark:hover:bg-gray-800 hover:cursor-pointer rounded-md justify-between w-full gap-2 overflow-hidden whitespace-nowrap ring-0 focus:ring-0 focus:outline-none ">
        <div className="flex items-center gap-2 whitespace-nowrap overflow-hidden">
          <Avatar className="w-6 h-6">
            {currentOrg?.imageUrl ? (
              <AvatarImage src={currentOrg.imageUrl} alt={currentOrg.name ?? "Profile picture"} />
            ) : user?.imageUrl ? (
              <AvatarImage
                src={user.imageUrl}
                alt={user?.username ?? user?.fullName ?? "Profile picture"}
              />
            ) : null}
            <AvatarFallback className="flex h-8 w-8 items-center justify-center rounded border border-gray-500 bg-gray-100 text-gray-700">
              {(currentOrg?.name ?? user?.username ?? user?.fullName ?? "")
                .slice(0, 2)
                .toUpperCase() ?? "P"}
            </AvatarFallback>
          </Avatar>
          {!isLoaded ? (
            <Loading />
          ) : (
            <Tooltip>
              <TooltipTrigger className="w-full overflow-hidden text-ellipsis">
                <span className="overflow-hidden text-ellipsis text-sm font-semibold">
                  {currentOrg?.name ?? "Personal Workspace"}
                </span>
              </TooltipTrigger>
              <TooltipContent>
                <span className="text-sm font-semibold">
                  {currentOrg?.name ?? "Personal Workspace"}
                </span>
              </TooltipContent>
            </Tooltip>
          )}
        </div>

        <ChevronsUpDown className="hidden h-3 w-3 shrink-0 md:block" />
      </DropdownMenuTrigger>
      <DropdownMenuContent
        side="left"
        className="w-96 absolute left-0 lg:left-8 top-8 max-sm:left-0"
      >
        <DropdownMenuLabel>Personal Account</DropdownMenuLabel>
        <DropdownMenuItem
          className="flex items-center justify-between"
          onClick={() => changeOrg(null)}
        >
          <span className={currentOrg === null ? "font-semibold" : undefined}>
            {user?.username ?? user?.fullName ?? "Personal Workspace"}
          </span>
          {currentOrg === null ? <Check className="h-4 w-4" /> : null}
        </DropdownMenuItem>
        <DropdownMenuSeparator />

        <DropdownMenuLabel>Workspaces</DropdownMenuLabel>
        <DropdownMenuGroup>
          {filteredOrgs.map((membership) => (
            <DropdownMenuItem
              key={membership.id}
              className="flex items-center justify-between"
              onClick={() => changeOrg(membership.organization.id)}
            >
              <span
                className={
                  membership.organization.id === currentOrg?.id ? "font-semibold" : undefined
                }
              >
                {" "}
                {membership.organization.name}
              </span>
              {membership.organization.id === currentOrg?.id ? <Check className="h-4 w-4" /> : null}
            </DropdownMenuItem>
          ))}
          <DropdownMenuSeparator />

          <DropdownMenuItem>
            <Link href="/new" className="flex items-center">
              <Plus className="mr-2 h-4 w-4" />
              <span>Create Workspace</span>
            </Link>
          </DropdownMenuItem>
          {membership?.role === "admin" ? (
            <Link href="/app/settings/team">
              <DropdownMenuItem>
                <UserPlus className="w-4 h-4 mr-2 " />
                <span className="cursor-pointer">Invite Member</span>
              </DropdownMenuItem>
            </Link>
          ) : null}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
