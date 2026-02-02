"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { redirect } from "next/navigation";

export default function BillingPage() {
  const workspace = useWorkspaceNavigation();

  // Redirect to connect page by default
  redirect(`/${workspace.slug}/billing/connect`);
}
