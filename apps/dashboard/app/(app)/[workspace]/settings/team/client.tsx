"use client";

import { PageHeader } from "@/components/dashboard/page-header";
import { trpc } from "@/lib/trpc/client";
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
import { useMemo, useState } from "react";
import { Invitations } from "./invitations";
import { InviteButton } from "./invite";
import { Members } from "./members";

export function TeamPageClient({ team, workspaceId }: { team: boolean; workspaceId: string }) {
  const { data: user } = trpc.user.getCurrentUser.useQuery();

  const { data: memberships, isLoading: isUserMembershipsLoading } =
    trpc.user.listMemberships.useQuery(user?.id || "", {
      enabled: !!user,
    });

  const { data: organization, isLoading: isOrganizationLoading } = trpc.org.getOrg.useQuery(
    user?.orgId || "",
    {
      enabled: !!user,
    },
  );

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
            <Link href={`/${workspaceId}/settings/billing`}>
              <Button>Upgrade</Button>
            </Link>
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
