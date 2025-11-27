"use client";;
import { PageHeader } from "@/components/dashboard/page-header";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { useTRPC } from "@/lib/trpc/client";
import {
  Button,
  Empty,
  Loading,
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import Link from "next/link";
import { Suspense, useMemo, useState } from "react";
import { Invitations } from "./invitations";
import { InviteButton } from "./invite";
import { Members } from "./members";

import { useQuery } from "@tanstack/react-query";

export function TeamPageClient({ team }: { team: boolean }) {
  const trpc = useTRPC();
  const workspace = useWorkspaceNavigation();

  if (!workspace) {
    return null;
  }

  const { data: user } = useQuery(trpc.user.getCurrentUser.queryOptions());

  const { data: memberships, isLoading: isUserMembershipsLoading } =
    useQuery(trpc.user.listMemberships.queryOptions(user?.id || "", {
      enabled: !!user,
    }));

  const { data: organization, isLoading: isOrganizationLoading } = useQuery(trpc.org.getOrg.queryOptions(
    user?.orgId || "",
    {
      enabled: !!user,
    },
  ));

  const userMemberships = memberships?.data;

  const currentOrgMembership = userMemberships?.find(
    (membership) => membership.organization.id === user?.orgId,
  );

  const isAdmin = useMemo(() => {
    return user?.role === "admin";
  }, [user?.role]);

  const isLoading = useMemo(() => {
    return isUserMembershipsLoading || isOrganizationLoading || !user;
  }, [isUserMembershipsLoading, isOrganizationLoading, user]);

  type Tab = "members" | "invitations";
  const [tab, setTab] = useState<Tab>("members");

  // make typescript happy
  if (!user || !organization || !userMemberships || !currentOrgMembership) {
    return null;
  }

  const actions: React.ReactNode[] = [];

  if (isAdmin) {
    actions.push(
      <Select key="tab-select" value={tab} onValueChange={(value: Tab) => setTab(value)}>
        <SelectTrigger className="w-[180px]">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            <SelectItem value="members">Members</SelectItem>
            <SelectItem value="invitations">Invitations</SelectItem>
          </SelectGroup>
        </SelectContent>
      </Select>,
    );

    actions.push(<InviteButton key="invite-button" user={user} organization={organization} />);
  }

  if (!team) {
    return (
      <div className="relative items-center justify-center h-screen w-full">
        <Empty className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-[60%] w-full">
          <Empty.Title>Upgrade Your Plan to Add Team Members</Empty.Title>
          <Empty.Description>You can try it out for free for 14 days.</Empty.Description>
          <Empty.Actions>
            <Suspense fallback={<Loading type="spinner" />}>
              <Link href={`/${workspace.slug}/settings/billing`}>
                <Button>Upgrade</Button>
              </Link>
            </Suspense>
          </Empty.Actions>
        </Empty>
      </div>
    );
  }

  return (
    <>
      <PageHeader title="Members" description="Manage your team members" actions={actions} />
      {isLoading || !user || !organization || !userMemberships || !currentOrgMembership ? (
        <Loading />
      ) : tab === "members" ? (
        <Members organization={organization} user={user} userMembership={currentOrgMembership} />
      ) : (
        <Invitations organization={organization} user={user} />
      )}
    </>
  );
}
