"use client";

import { useWorkspace } from "@/providers/workspace-provider";
import * as flags from ".";

export function useFlag(key: keyof typeof flags): boolean {
  const { workspace } = useWorkspace();
  const flag = flags[key];
  return workspace?.flags[flag.slug] ?? flag.defaultValue;
}
