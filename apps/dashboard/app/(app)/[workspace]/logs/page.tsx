"use client";
import { Navigation } from "@/components/navigation/navigation";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { Layers3 } from "@unkey/icons";
import { useRouter } from "next/navigation";
import { LogsClient } from "./components/logs-client";
import { redirect } from "next/navigation";
import { Loading } from "@unkey/ui";
export const dynamic = "force-dynamic";

export default function Page() {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();

  return (
    <div>
      <Navigation
        href={`/${workspace.slug}/logs`}
        name="Logs"
        icon={<Layers3 />}
      />
      <LogsClient />
    </div>
  );
}
