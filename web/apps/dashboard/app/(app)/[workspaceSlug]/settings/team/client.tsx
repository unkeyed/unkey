"use client";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { Button, Empty, Loading } from "@unkey/ui";
import Link from "next/link";
import { Suspense, useMemo, useState } from "react";
import { Invitations } from "./invitations";
import { InviteForm } from "./invite-form";
import { Members } from "./members";

export function TeamPageClient({ team }: { team: boolean }) {
  const workspace = useWorkspaceNavigation();

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
    <div className="flex flex-col gap-8 w-full">
      <div className="flex flex-col gap-2">
        <h1 className="text-2xl font-semibold text-content">Members</h1>
        <p className="text-sm text-content-subtle">Manage team members and invitations</p>
      </div>

      {isAdmin && <InviteForm organization={organization} />}

      <Tabs value={tab} onValueChange={(value) => setTab(value as Tab)} className="w-full">
        <TabsList className="inline-flex h-auto items-center justify-start bg-transparent p-0 border-b border-border w-full">
          <TabsTrigger
            value="members"
            className="rounded-none border-b-2 border-transparent px-4 py-2 data-[state=active]:bg-transparent data-[state=active]:border-content data-[state=active]:shadow-none"
          >
            Team Members
          </TabsTrigger>
          <TabsTrigger
            value="invitations"
            className="rounded-none border-b-2 border-transparent px-4 py-2 data-[state=active]:bg-transparent data-[state=active]:border-content data-[state=active]:shadow-none"
          >
            Pending Invitations
          </TabsTrigger>
        </TabsList>

        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <Loading />
          </div>
        ) : (
          <>
            <TabsContent value="members" className="mt-6 min-h-[400px]">
              <Members
                organization={organization}
                user={user}
                userMembership={currentOrgMembership}
              />
            </TabsContent>
            <TabsContent value="invitations" className="mt-6 min-h-[400px]">
              <Invitations organization={organization} />
            </TabsContent>
          </>
        )}
      </Tabs>
    </div>
  );
}
