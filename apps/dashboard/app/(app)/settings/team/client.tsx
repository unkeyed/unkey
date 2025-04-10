"use client";

import { Loading } from "@/components/dashboard/loading";
import { PageHeader } from "@/components/dashboard/page-header";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { trpc } from "@/lib/trpc/client";
import { Empty } from "@unkey/ui";
import { Button } from "@unkey/ui";
import Link from "next/link";
import { useMemo, useState } from "react";
import { Invitations } from "./invitations";
import { InviteButton } from "./invite";
import { Members } from "./members";

export default function TeamPageClient({ team }: { team: boolean }) {
  const { data: user } = trpc.user.getCurrentUser.useQuery();
  const { data: memberships, isLoading: isUserMembershipsLoading } =
    trpc.user.listMemberships.useQuery(user?.id as string, {
      enabled: !!user,
    });
  const { data: organization, isLoading: isOrganizationLoading } = trpc.org.getOrg.useQuery(
    user?.orgId! as string,
    {
      enabled: !!user,
    },
  );
  const userMemberships = memberships?.data;
  const currentOrgMembership = userMemberships?.find(
    (membership) => membership.organization.id === user?.orgId,
  );

  const isAdmin = useMemo(() => {
    return currentOrgMembership?.role === "admin";
  }, [currentOrgMembership]);

  const isLoading = useMemo(() => {
    return isUserMembershipsLoading || isOrganizationLoading;
  }, [isUserMembershipsLoading, isOrganizationLoading]);

  type Tab = "members" | "invitations";
  const [tab, setTab] = useState<Tab>("members");

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

    actions.push(<InviteButton key="invite-button" user={user!} organization={organization!} />);
  }

  if (!team) {
    return (
      <div className="w-full h-screen -mt-40 flex items-center justify-center">
        <Empty>
          <Empty.Title>Upgrade Your Plan to Add Team Members</Empty.Title>
          <Empty.Description>You can try it out for free for 14 days.</Empty.Description>
          <Empty.Actions>
            <Link href="/settings/billing">
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
      {isLoading ? (
        <Loading />
      ) : tab === "members" ? (
        <Members organization={organization!} user={user!} userMembership={currentOrgMembership!} />
      ) : (
        <Invitations organization={organization!} user={user!} />
      )}
    </>
  );
}
