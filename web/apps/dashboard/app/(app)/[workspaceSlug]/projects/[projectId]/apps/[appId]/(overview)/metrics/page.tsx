"use client";

import { useFlag } from "@/lib/flags/provider";
import { notFound } from "next/navigation";
import { MetricsPage } from "./metrics-page";

export default function Page() {
  if (!useFlag("appV2")) {
    notFound();
  }
  return <MetricsPage />;
}
