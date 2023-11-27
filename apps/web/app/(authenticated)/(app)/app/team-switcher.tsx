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
  const { setActive, organizationList, isLoaded: clerkLoaded } = useOrganizationList();
  const { organization: currentOrg, membership } = useOrganization();
  const { user } = useUser();
  const [isLoading, setLoading] = useState(false);
  const _router = useRouter();
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
      _router.refresh();
    }
  }
  const [search, _setSearch] = useState("");
  const filteredOrgs = useMemo(() => {
    if (!organizationList) {
      return [];
    }
    if (search === "") {
      return organizationList;
    }
    return organizationList?.filter(({ organization }) =>
      organization.name.toLowerCase().includes(search.toLowerCase()),
    );
  }, [search, organizationList])!;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="flex items-center px-2 hover:bg-gray-200 dark:hover:bg-gray-800 hover:cursor-pointer rounded-md justify-between w-full h-full gap-2 overflow-hidden whitespace-nowrap ring-0 focus:ring-0 focus:outline-none ">
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
          {!clerkLoaded || isLoading ? (
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
      <DropdownMenuContent side="right" className="w-96">
        <DropdownMenuLabel>Personal Account</DropdownMenuLabel>
        <DropdownMenuItem
          className="flex items-center justify-between"
          onClick={() => changeOrg(null)}
        >
          <span className={currentOrg === null ? "font-semibold" : undefined}>
            {user?.username ?? user?.fullName ?? ""}
          </span>
          {currentOrg === null ? <Check className="h-4 w-4" /> : null}
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
              {organization.id === currentOrg?.id ? <Check className="h-4 w-4" /> : null}
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
            <Link href="/app/team/invite">
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
