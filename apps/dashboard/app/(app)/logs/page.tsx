"use client";
import { Layers3 } from "@unkey/icons";
import { LogsClient } from "./components/logs-client";

import { Navigation } from "@/components/navigation/navigation";

export default function Page() {
  return (
    <div>
      <Navigation href="/logs" name="Logs" icon={<Layers3 />} />
      <LogsClient />
    </div>
  );
}
