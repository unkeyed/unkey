"use client";
import { OptIn } from "@/components/opt-in";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { IdentitiesClient } from "./_components/identities-client";
import { Navigation } from "./navigation";
export const dynamic = "force-dynamic";

export default function Page() {
  const workspace = useWorkspaceNavigation();

  if (!workspace.betaFeatures.identities) {
    return <OptIn title="Identities" description="Identities are in beta" feature="identities" />;
  }

  return (
    <div>
      <Navigation />
      <IdentitiesClient />
    </div>
  );
}
