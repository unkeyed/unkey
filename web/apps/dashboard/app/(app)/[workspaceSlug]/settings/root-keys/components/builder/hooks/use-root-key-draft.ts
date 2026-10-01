"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { listRootKeys, rootKeysV2QueryKeys } from "@/lib/root-keys-api";
import { useQuery } from "@tanstack/react-query";
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
  const { data, error } = useQuery({
    queryKey: rootKeysV2QueryKeys.workspace(workspace.id),
    queryFn: ({ signal }) => listRootKeys(signal),
    enabled: isOpen,
    retry: false,
    select: (rootKeys) => rootKeys.find((rootKey) => rootKey.keyId === keyId) ?? null,
  });
  const message =
    error instanceof Error ? error.message : error ? "Failed to load the Root Key." : null;

  useEffect(() => {
    if (message === null) {
      return;
    }
    toast.error(message);
    onUnavailable();
  }, [message, onUnavailable]);

  useEffect(() => {
    if (data !== null) {
      return;
    }
    toast.error("Root Key not found. It may have been deleted.");
    onUnavailable();
  }, [data, onUnavailable]);

  return useMemo(() => {
    if (data === undefined || data === null) {
      return null;
    }
    const name = data.name ?? "";
    const { policies, unmapped } = grantsToPolicies(workspace.id, data.permissions);
    if (unmapped.length > 0) {
      return {
        kind: "legacy",
        keyId: data.keyId,
        start: data.start,
        name,
        grants: data.permissions,
      };
    }
    return { kind: "editable", keyId: data.keyId, start: data.start, name, policies };
  }, [data, workspace.id]);
}
