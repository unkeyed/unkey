"use client";

import { useWorkspace } from "@/providers/workspace-provider";
import * as flags from ".";
import { useFlag as resolveFlag } from "../workspace-flags";

export function useFlag(key: keyof typeof flags): boolean {
  const { workspace } = useWorkspace();
  return resolveFlag({ workspace, flag: flags[key] });
}
