import { deployAnomalyAlerts } from "@/lib/flags";
import { notFound } from "next/navigation";
import type { ReactNode } from "react";

export default async function AlertsLayout({ children }: { children: ReactNode }) {
  if (!(await deployAnomalyAlerts())) {
    notFound();
  }

  return children;
}
