"use client";

import { useDatabasesEnabled } from "@/lib/flags/use-databases-enabled";
import { notFound } from "next/navigation";
import type { ReactNode } from "react";

export default function DatabasesLayout({ children }: { children: ReactNode }) {
  if (!useDatabasesEnabled()) {
    notFound();
  }
  return children;
}
