import { routes } from "@/lib/navigation/routes";
import { redirect } from "next/navigation";

// Root Keys moved to the top-level workspace sidebar; keep the old settings
// URL alive by redirecting to the canonical /root-keys path.
export default async function LegacyRootKeysPage({
  params,
}: {
  params: Promise<{ workspaceSlug: string }>;
}) {
  const { workspaceSlug } = await params;
  redirect(routes.rootKeys.list({ workspaceSlug }));
}
