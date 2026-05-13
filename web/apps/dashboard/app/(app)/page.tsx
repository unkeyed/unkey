"use client";

import { useFlag } from "@/lib/flags/provider";
import { useWorkspace } from "@/providers/workspace-provider";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

export default function AppHomePage() {
  const { workspace } = useWorkspace();
  const router = useRouter();
  const newNavigation = useFlag("newNavigation");

  useEffect(() => {
    if (workspace) {
      const home = newNavigation ? "projects" : "apis";
      router.push(`/${workspace.slug}/${home}`);
    }
  }, [workspace, router, newNavigation]);

  return null; // Layout handles loading states
}
