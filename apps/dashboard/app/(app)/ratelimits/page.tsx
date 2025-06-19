"use client";

import { RatelimitClient } from "./_components/ratelimit-client";
import { Navigation } from "./navigation";

export default function RatelimitOverviewPage() {
  return (
    <div>
      <Navigation />
      <RatelimitClient />
    </div>
  );
}
