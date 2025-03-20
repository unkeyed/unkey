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
import { ScrollArea } from "@/components/ui/scroll-area";
import { useSidebar } from "@/components/ui/sidebar";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { useOrganization, useOrganizationList, useUser } from "@clerk/nextjs";
import { ChevronExpandY } from "@unkey/icons";
import { Check, Plus, UserPlus } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import type React from "react";
import { useMemo, useState } from "react";

type Props = {
  workspace: {
    name: string;
  };
};

export const WorkspaceSwitcher: React.FC<Props> = (props): JSX.Element => {
  const { isLoaded, setActive, userMemberships } = useOrganizationList({
    userMemberships: {
      infinite: true,
      pageSize: 100,
    },
  });
  const { organization: currentOrg, membership } = useOrganization();
  const { user } = useUser();
  const router = useRouter();
  const { isMobile, state } = useSidebar();

  // Only collapsed in desktop mode, not in mobile mode
  const isCollapsed = state === "collapsed" && !isMobile;

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
      <DropdownMenuTrigger
        className={cn(
          "flex items-center bg-base-12 overflow-hidden rounded-lg bg-background border-gray-6 border hover:bg-background-subtle hover:cursor-pointer whitespace-nowrap ring-0 focus:ring-0 focus:outline-none text-content",
          isCollapsed ? "justify-center w-10 h-8 p-0" : "justify-between w-full h-8 gap-2 px-2",
        )}
      >
        <div
          className={cn(
            "flex items-center gap-2 overflow-hidden whitespace-nowrap",
            isCollapsed ? "justify-center" : "",
          )}
        >
          <Avatar className="w-5 h-5 rounded border border-grayA-6">
            {currentOrg?.imageUrl ? (
              <AvatarImage src={currentOrg.imageUrl} alt={props.workspace.name} />
            ) : user?.imageUrl ? (
              <AvatarImage
                src={user.imageUrl}
                alt={user?.username ?? user?.fullName ?? "Profile picture"}
              />
            ) : null}
            <AvatarFallback className="flex items-center justify-center w-8 h-8 text-gray-700 bg-gray-100 border border-gray-500 rounded">
              {props.workspace.name.slice(0, 2).toUpperCase()}
            </AvatarFallback>
          </Avatar>
          {!isLoaded ? (
            <Loading />
          ) : !isCollapsed ? (
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
          ) : null}
        </div>
        {!isCollapsed && (
          <ChevronExpandY className="w-5 h-5 shrink-0 md:block [stroke-width:1px] text-gray-9" />
        )}
      </DropdownMenuTrigger>

      <DropdownMenuContent
        className="absolute left-0 w-72 lg:w-96 max-sm:left-0 bg-gray-1 dark:bg-black shadow-2xl border-gray-6 rounded-lg"
        align="start"
      >
        <DropdownMenuLabel>Personal Account</DropdownMenuLabel>
        <DropdownMenuItem
          className="flex items-center justify-between"
          onClick={() => changeOrg(null)}
        >
          <span className={currentOrg === null ? "font-medium" : undefined}>
            {user?.username ?? user?.fullName ?? "Personal Workspace"}
          </span>
          {currentOrg === null ? <Check className="w-4 h-4" /> : null}
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuLabel>Workspaces</DropdownMenuLabel>
        <DropdownMenuGroup>
          <ScrollArea className="h-96">
            {filteredOrgs.map((membership) => (
              <DropdownMenuItem
                key={membership.id}
                className="flex items-center justify-between"
                onClick={() => changeOrg(membership.organization.id)}
              >
                <span
                  className={
                    membership.organization.id === currentOrg?.id ? "font-medium" : undefined
                  }
                >
                  {" "}
                  {membership.organization.name}
                </span>
                {membership.organization.id === currentOrg?.id ? (
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
          {membership?.role === "admin" ? (
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
