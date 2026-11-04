"use client";

import { useConnectionsEnabled } from "@/lib/flags/use-connections-enabled";
import { notFound } from "next/navigation";
import { ConnectionsPage } from "./connections-page";

export default function Page() {
  if (!useConnectionsEnabled()) {
    notFound();
  }
  return <ConnectionsPage />;
}
