/**
 * Deprecated with new auth
 * Hiding for now until we decide if we want to fix it up or toss it
 */

"use client";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Check, ChevronExpandY } from "@unkey/icons";
import type React from "react";
import { useState } from "react";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { setSessionCookie } from "@/lib/auth/cookies";
import { useTRPC } from "@/lib/trpc/client";
import { Loading, toast } from "@unkey/ui";

import { useQuery } from "@tanstack/react-query";
import { useMutation } from "@tanstack/react-query";
import { useQueryClient } from "@tanstack/react-query";

export const WorkspaceSwitcher: React.FC = (): JSX.Element => {
  const trpc = useTRPC();
  const { data: user } = useQuery(trpc.user.getCurrentUser.queryOptions());
  const { data: memberships, isLoading: isUserMembershipsLoading } = useQuery(
    trpc.user.listMemberships.queryOptions(
      user?.id as string, // make typescript happy
      {
        enabled: !!user,
      },
    ),
  );
  const queryClient = useQueryClient();
  // const { switchOrganization } = useUser();
  // const { organization: currentOrg } = useOrganization();
  const [isLoading, setLoading] = useState(false);
  if (!memberships?.data) {
    console.error("Memberships data is not available");
    return <div>Unable to load workspace data</div>; // or appropriate fallback UI
  }

  const userMemberships = memberships.data;
  const currentOrg = userMemberships.find(
    (membership) => membership.organization.id === user?.orgId,
  );

  const changeWorkspace = useMutation(
    trpc.user.switchOrg.mutationOptions({
      async onSuccess(sessionData) {
        if (!sessionData.token || !sessionData.expiresAt) {
          console.error("Invalid session data received:", sessionData);
          toast.error("Failed to switch workspace. Invalid session data.");
          return;
        }

        try {
          await setSessionCookie({
            token: sessionData.token,
            expiresAt: sessionData.expiresAt,
          });
          queryClient.invalidateQueries(trpc.user.getCurrentUser.pathFilter());
        } catch (error) {
          console.error("Failed to set session cookie:", error);
          toast.error("Failed to complete workspace switch. Please try again.");
        }
      },
      onError(error) {
        console.error("Failed to switch workspace: ", error);
        toast.error("Failed to switch workspace. Contact support if error persists.");
      },
    }),
  );

  async function changeOrg(orgId: string | null) {
    if (!orgId) {
      return;
    }
    try {
      setLoading(true);
      changeWorkspace.mutateAsync(orgId);
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
              <AvatarImage src={user.avatarUrl} alt={user?.fullName ?? "Profile picture"} />
            ) : null}
            <AvatarFallback className="flex items-center justify-center w-8 h-8 overflow-hidden text-gray-700 bg-gray-100 border border-gray-500 rounded">
              {(user?.fullName ?? "").slice(0, 1).toUpperCase() ?? "P"}
            </AvatarFallback>
          </Avatar>
          {isUserMembershipsLoading || isLoading ? (
            <Loading />
          ) : (
            <span className="text-sm font-semibold">
              {currentOrg?.organization.name ?? "Free Workspace"}
            </span>
          )}
        </div>
        <ChevronExpandY className="hidden w-3 h-3 md:block" />
      </DropdownMenuTrigger>
      <DropdownMenuContent side="right" className="w-96">
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
                  membership.organization.id === currentOrg?.organization.id
                    ? "font-semibold"
                    : undefined
                }
              >
                {" "}
                {membership.organization.name}
              </span>
              {membership.organization.id === currentOrg?.organization.id ? (
                <Check className="w-4 h-4" />
              ) : null}
            </DropdownMenuItem>
          ))}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
};
