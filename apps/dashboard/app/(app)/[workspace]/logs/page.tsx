"use client";
import { Navigation } from "@/components/navigation/navigation";
import { useWorkspace } from "@/providers/workspace-provider";
import { Layers3 } from "@unkey/icons";
import { useRouter } from "next/navigation";
import { LogsClient } from "./components/logs-client";
export const dynamic = "force-dynamic";

export default async function Page() {
  const router = useRouter();
  const { workspace, isLoading } = useWorkspace();

  if (!workspace && !isLoading) {
    return router.push("/new");
  }

  return (
    <div>
      <Navigation href={`/${workspace?.slug}/logs`} name="Logs" icon={<Layers3 />} />
      <LogsClient />
    </div>
  );
}
