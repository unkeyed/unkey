"use client";

import { Loading } from "@/components/dashboard/loading";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
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
import { Check, Plus, UserPlus } from "lucide-react";
import { useRouter } from "next/navigation";
import type React from "react";
import { useMemo, useState } from "react";
import { ScrollArea } from "@/components/ui/scroll-area";
import { ChevronExpandY } from "@unkey/icons";
import Link from "next/link";
import { useUser } from "@/lib/auth/hooks";

type Props = {
  workspace: {
    name: string;
  };
};
export const WorkspaceSwitcher: React.FC<Props> = (props): JSX.Element => {
  const { user } = useUser();
  const router = useRouter();
  const { memberships: userMemberships, switchOrganization, loading } = useUser();
  const currentOrgMembership = userMemberships.find(
    (membership) => membership.organization.id === user?.orgId,
  );

  async function changeWorkspace(orgId: string | null) {
    try {
      if (!orgId) {
        return;
      }
      await switchOrganization(orgId);
    } finally {
      router.refresh();
    }
  }

  const [search, _setSearch] = useState("");
  const filteredOrgs = useMemo(() => {
    if (!userMemberships || userMemberships.length === 0) {
      return [];
    }
    if (search === "") {
      return userMemberships;
    }
    return userMemberships.filter((m) =>
      m.organization.name.toLowerCase().includes(search.toLowerCase()),
    );
  }, [search, userMemberships])!;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="flex items-center justify-between w-full h-10 gap-2 px-2 overflow-hidden rounded-[0.625rem] bg-background border-border border hover:bg-background-subtle hover:cursor-pointer whitespace-nowrap ring-0 focus:ring-0 focus:outline-none text-content">
        <div className="flex items-center gap-2 overflow-hidden whitespace-nowrap">
          <Avatar className="w-5 h-5">
            {user?.avatarUrl ? (
              <AvatarImage src={user.avatarUrl} alt={user?.fullName ?? "Profile picture"} />
            ) : null}
            <AvatarFallback className="flex items-center justify-center w-8 h-8 text-gray-700 bg-gray-100 border border-gray-500 rounded">
              {props.workspace.name.slice(0, 2).toUpperCase()}
            </AvatarFallback>
          </Avatar>
          {loading.memberships ? (
            <Loading />
          ) : (
            <Tooltip>
              <TooltipTrigger asChild>
                <span className="overflow-hidden text-sm font-medium text-ellipsis">
                  {props.workspace.name}
                </span>
              </TooltipTrigger>
              <TooltipContent>
                <span className="text-sm font-medium">{props.workspace.name}</span>
              </TooltipContent>
            </Tooltip>
          )}
        </div>

        <ChevronExpandY className="hidden w-5 h-5 shrink-0 md:block [stroke-width:1px]" />
      </DropdownMenuTrigger>
      <DropdownMenuContent className="absolute left-0 w-96 max-sm:left-0">
        <DropdownMenuLabel>Workspaces</DropdownMenuLabel>
        <DropdownMenuGroup>
          <ScrollArea className="h-96">
            {filteredOrgs.map((membership) => (
              <DropdownMenuItem
                key={membership.id}
                className="flex items-center justify-between"
                onClick={() => changeWorkspace(membership.organization.id)}
              >
                <span
                  className={
                    membership.organization.id === currentOrgMembership?.organization.id
                      ? "font-medium"
                      : undefined
                  }
                >
                  {" "}
                  {membership.organization.name}
                </span>
                {membership.organization.id === currentOrgMembership?.organization.id ? (
                  <Check className="w-4 h-4" />
                ) : null}
              </DropdownMenuItem>
            ))}
          </ScrollArea>
          <DropdownMenuSeparator />

          <DropdownMenuItem>
            <Link href="/new" className="flex items-center">
              <Plus className="w-4 h-4 mr-2" />
              <span>Create Workspace</span>
            </Link>
          </DropdownMenuItem>
          {currentOrgMembership?.role === "admin" ? (
            <Link href="/settings/team">
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
