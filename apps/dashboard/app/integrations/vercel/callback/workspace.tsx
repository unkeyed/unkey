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
import type React from "react";
import { useState } from "react";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { useUser, useOrganization } from "@/lib/auth/hooks";

export const WorkspaceSwitcher: React.FC = (): JSX.Element => {
  const { switchOrganization, memberships: userMemberships, isLoading: loading, user } = useUser();
  const { organization: currentOrg } = useOrganization();
  const [isLoading, setLoading] = useState(false);
  async function changeOrg(orgId: string | null) {
    if (!orgId) {
      return;
    }
    try {
      setLoading(true);
      await switchOrganization(orgId);
    } finally {
      setLoading(false);
    }
  }
  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="flex items-center justify-between w-full gap-2">
        <div className="flex items-center gap-2">
          <Avatar className="w-6 h-6">
            {user?.avatarUrl ? (
              <AvatarImage
                src={user.avatarUrl}
                alt={user?.fullName ?? "Profile picture"}
              />
            ) : null}
            <AvatarFallback className="flex items-center justify-center w-8 h-8 overflow-hidden text-gray-700 bg-gray-100 border border-gray-500 rounded">
              {(user?.fullName ?? "")
                .slice(0, 2)
                .toUpperCase() ?? "P"}
            </AvatarFallback>
          </Avatar>
          {loading || isLoading ? (
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
            {user?.fullName ?? ""}
          </span>
          {currentOrg === null ? <Check className="w-4 h-4" /> : null}
        </DropdownMenuItem>
        <DropdownMenuSeparator />

        <DropdownMenuLabel>Workspaces</DropdownMenuLabel>
        <DropdownMenuGroup>
          {userMemberships?.map((membership) => (
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
