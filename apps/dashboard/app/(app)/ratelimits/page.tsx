"use client";

import { NamespaceListClient } from "./_components/namespace-list-client";
import { Navigation } from "./navigation";

export default function RatelimitOverviewPage() {
  return (
    <div>
      <Navigation />
      <NamespaceListClient />
    </div>
  );
}
