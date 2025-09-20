"use client";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
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
import { setSessionCookie } from "@/lib/auth/cookies";
import { reset } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { ChevronExpandY } from "@unkey/icons";
import { InfoTooltip, Loading, toast } from "@unkey/ui";
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
  const router = useRouter();
  const utils = trpc.useUtils();
  const { isMobile, state } = useSidebar();

  // Only collapsed in desktop mode, not in mobile mode
  const isCollapsed = state === "collapsed" && !isMobile;

  const { data: user } = trpc.user.getCurrentUser.useQuery();
  const { data: memberships, isLoading: isUserMembershipsLoading } =
    trpc.user.listMemberships.useQuery(
      user?.id as string, // make typescript happy
      {
        enabled: !!user,
      },
    );

  const userMemberships = memberships?.data;

  const currentOrgMembership = userMemberships?.find(
    (membership) => membership.organization.id === user?.orgId,
  );

  const changeWorkspace = trpc.user.switchOrg.useMutation({
    async onSuccess(sessionData) {
      if (!sessionData.token || !sessionData.expiresAt) {
        toast.error("Failed to switch workspace. Invalid session data.");
        return;
      }

      try {
        await setSessionCookie({
          token: sessionData.token,
          expiresAt: sessionData.expiresAt,
        });

        // refresh the check mark by invalidating the current user's org data
        utils.user.getCurrentUser.invalidate();
        utils.api.overview.query.invalidate();

        await reset();

        // reload data
        router.replace("/");
      } catch (error) {
        console.error("Failed to set session cookie:", error);
        toast.error("Failed to complete workspace switch. Please try again.");
      }
    },
    onError(error) {
      console.error("Failed to switch workspace: ", error);
      toast.error("Failed to switch workspace. Contact support if error persists.");
    },
  });
  const handleWorkspaceSwitch = async (targetOrgId: string) => {
    // Prevent switch if already on the target workspace
    if (targetOrgId === currentOrgMembership?.organization.id) {
      return;
    }

    await changeWorkspace.mutateAsync(targetOrgId);
  };

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
  }, [search, userMemberships]);

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
            <AvatarFallback className="text-gray-700 bg-gray-100 border border-gray-500 rounded">
              {props.workspace.name.slice(0, 1).toUpperCase()}
            </AvatarFallback>
          </Avatar>
          {isUserMembershipsLoading ? (
            <Loading type="dots" size={24} />
          ) : isCollapsed ? null : (
            <InfoTooltip
              variant="inverted"
              position={{ side: "right", sideOffset: 10 }}
              content={<span>{props.workspace.name}</span>}
              className="text-xs font-medium py-2"
              triggerClassName="overflow-hidden text-sm font-medium text-ellipsis"
            >
              {props.workspace.name}
            </InfoTooltip>
          )}
        </div>
        {!isCollapsed && (
          <ChevronExpandY className="w-5 h-5 shrink-0 md:block [stroke-width:1px] text-gray-9" />
        )}
      </DropdownMenuTrigger>

      <DropdownMenuContent
        className="absolute left-0 w-72 lg:w-96 max-sm:left-0 bg-gray-1 dark:bg-black shadow-2xl border-gray-6 rounded-lg"
        align="start"
      >
        <DropdownMenuLabel>Workspaces</DropdownMenuLabel>
        <DropdownMenuGroup>
          <ScrollArea className="h-96">
            {filteredOrgs.map((membership) => (
              <DropdownMenuItem
                key={membership.id}
                className="flex items-center justify-between"
                onClick={() => handleWorkspaceSwitch(membership.organization.id)}
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
