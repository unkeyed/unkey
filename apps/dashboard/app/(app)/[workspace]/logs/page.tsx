"use client";
import { Navigation } from "@/components/navigation/navigation";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Layers3 } from "@unkey/icons";
import { Loading } from "@unkey/ui";
import { Suspense } from "react";
import { LogsClient } from "./components/logs-client";
export const dynamic = "force-dynamic";

export default function Page() {
  const workspace = useWorkspaceNavigation();

  return (
    <div>
      <Suspense fallback={<Loading type="spinner" />}>
        <Navigation href={`/${workspace.slug}/logs`} name="Logs" icon={<Layers3 />} />
      </Suspense>
      <LogsClient />
    </div>
  );
}
