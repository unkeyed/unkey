import { getTenantId } from "@/lib/auth";
import type React from "react";
export default async function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  await getTenantId();

  return <>{children}</>;
}
