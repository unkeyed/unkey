import { redirect } from "next/navigation";
import type { Organization } from "@/lib/auth/types";
import { getLastUsedOrgCookie } from "@/lib/auth/cookies";
import { getUser } from "@/lib/auth/user";
import Page from "./page";

export default async function WorkspaceSelectionLayout() {
  const user = await getUser();

  if (!user) {
    redirect("/auth/sign-in");
  }

  // Fetch user's memberships
  // We need to get the memberships from the database
  // This is a simplified version - in reality, you'd fetch from the database
  const memberships = user.memberships || [];

  const organizations: Organization[] = memberships.map((membership) => ({
    id: membership.organization.id,
    name: membership.organization.name,
    slug: membership.organization.slug,
    createdAt: membership.organization.createdAt,
  }));

  const lastOrgId = await getLastUsedOrgCookie();

  // If user has only 1 workspace and has a last used workspace that matches, redirect to dashboard
  if (organizations.length === 1 && lastOrgId === organizations[0].id) {
    redirect(`/${organizations[0].slug}`);
  }

  // If user has no workspaces, show the page (it will show an empty state)
  // If user has multiple workspaces, show the page
  // If user has 1 workspace but no last used, show the page

  return <Page organizations={organizations} lastOrgId={lastOrgId} />;
}