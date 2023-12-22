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
import { Check, ChevronsUpDown } from "lucide-react";
import React, { useState } from "react";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { useOrganization, useOrganizationList, useUser } from "@clerk/nextjs";

export const WorkspaceSwitcher: React.FC = (): JSX.Element => {
  const { setActive, userMemberships, isLoaded: clerkLoaded } = useOrganizationList();
  const { organization: currentOrg } = useOrganization();
  const { user } = useUser();
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
  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="flex items-center justify-between w-full gap-2">
        <div className="flex items-center gap-2">
          <Avatar className="w-6 h-6">
            {currentOrg?.imageUrl ? (
              <AvatarImage src={currentOrg.imageUrl} alt={currentOrg.name ?? "Profile picture"} />
            ) : user?.imageUrl ? (
              <AvatarImage
                src={user.imageUrl}
                alt={user?.username ?? user?.fullName ?? "Profile picture"}
              />
            ) : null}
            <AvatarFallback className="flex items-center justify-center w-8 h-8 overflow-hidden text-gray-700 bg-gray-100 border border-gray-500 rounded">
              {(currentOrg?.name ?? user?.username ?? user?.fullName ?? "")
                .slice(0, 2)
                .toUpperCase() ?? "P"}
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
            {user?.username ?? user?.fullName ?? ""}
          </span>
          {currentOrg === null ? <Check className="w-4 h-4" /> : null}
        </DropdownMenuItem>
        <DropdownMenuSeparator />

        <DropdownMenuLabel>Workspaces</DropdownMenuLabel>
        <DropdownMenuGroup>
          {userMemberships?.data?.map((membership) => (
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
              {membership.organization.id === currentOrg?.id ? <Check className="w-4 h-4" /> : null}
            </DropdownMenuItem>
          ))}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
