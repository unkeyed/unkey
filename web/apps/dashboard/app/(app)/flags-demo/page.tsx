"use client";

import { useFlag } from "@/lib/flags/provider";

export default function FlagsDemoPage() {
  const enabled = useFlag("helloWorld");
  return (
    <div className="p-6">
      <h1 className="text-lg font-medium">Flags demo</h1>
      <p className="mt-2">hello-world: {enabled ? "on" : "off"}</p>
    </div>
  );
}
