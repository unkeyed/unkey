import { deletionRecoveryPage } from "@/lib/flags";
import { notFound } from "next/navigation";
import { ScheduledDeletionsClient } from "./client";

// Server-side flag gate. When the flag is off the route renders a 404
// so neither the URL nor the page content leaks to users who shouldn't
// see the recovery surface. The actual UI lives in [ScheduledDeletionsClient].
export default async function ScheduledDeletionsPage() {
  const enabled = await deletionRecoveryPage();
  if (!enabled) {
    notFound();
  }
  return <ScheduledDeletionsClient />;
}
