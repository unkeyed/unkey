"use client";

import { useFlag } from "@/lib/flags/provider";
import { notFound } from "next/navigation";
import { BuilderShell } from "./components/builder-shell";

export default function NewRootKeyPage() {
  const rootKeyBuilder = useFlag("rootKeyBuilder");

  if (!rootKeyBuilder) {
    notFound();
  }

  return <BuilderShell />;
}
