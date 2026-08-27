"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { toast } from "@unkey/ui";
import { useEffect, useMemo } from "react";
import type { Policy } from "../lib/policy";
import { grantsToPolicies } from "../lib/urn-parse";

export type EditableRootKeyDraft = {
  kind: "editable";
  keyId: string;
  start: string;
  name: string;
  policies: Policy[];
};

export type LegacyRootKeyDraft = {
  kind: "legacy";
  keyId: string;
  start: string;
  name: string;
  grants: string[];
};

export type RootKeyDraft = EditableRootKeyDraft | LegacyRootKeyDraft;

export function useRootKeyDraft(
  keyId: string,
  isOpen: boolean,
  onUnavailable: () => void,
): RootKeyDraft | null {
  const workspace = useWorkspaceNavigation();
  const { data, error } = trpc.settings.rootKeys.get.useQuery(
    { keyId },
    { enabled: isOpen, retry: false },
  );
  const message = error?.message ?? null;

  useEffect(() => {
    if (message === null) {
      return;
    }
    toast.error(message);
    onUnavailable();
  }, [message, onUnavailable]);

  return useMemo(() => {
    if (!data) {
      return null;
    }
    const name = data.name ?? "";
    const { policies, unmapped } = grantsToPolicies(workspace.id, data.grants);
    if (unmapped.length > 0) {
      return { kind: "legacy", keyId: data.id, start: data.start, name, grants: data.grants };
    }
    return { kind: "editable", keyId: data.id, start: data.start, name, policies };
  }, [data, workspace.id]);
}
