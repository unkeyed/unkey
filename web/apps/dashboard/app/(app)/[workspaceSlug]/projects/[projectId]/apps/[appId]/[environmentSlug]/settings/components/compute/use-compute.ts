"use client";

import { useWorkspace } from "@/providers/workspace-provider";
import { useMemo, useState } from "react";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateEnvironment } from "../../hooks/use-update-environment";
import { useReportUnsavedChanges } from "../../prevent-leave-context";
import {
  type ComputeDraft,
  type ComputeLimits,
  applyDraft,
  diffDraft,
  fromSettings,
  resolveLimits,
} from "./model";

export type SaveMode = "autosave" | "manual";

export type ComputePage = {
  base: ComputeDraft;
  limits: ComputeLimits;
  hovered: string | null;
  setHovered: (name: string | null) => void;
  commit: (next: ComputeDraft) => void;
  saveMode: SaveMode;
};

export type ComputeCard = ComputePage & {
  draft: ComputeDraft;
  edit: (fn: (current: ComputeDraft) => ComputeDraft) => void;
  dirty: boolean;
  changes: string[];
  save: () => void;
  discard: () => void;
};

export function useCompute(): ComputePage {
  const { settings, variant } = useEnvironmentSettings();
  const { limits } = useWorkspace();
  const updateEnvironment = useUpdateEnvironment();
  const [hovered, setHovered] = useState<string | null>(null);
  const base = useMemo(() => fromSettings(settings), [settings]);
  return {
    base,
    limits: resolveLimits(limits),
    hovered,
    setHovered,
    commit: (next) => updateEnvironment((target) => applyDraft(target, next)),
    saveMode: variant === "onboarding" ? "autosave" : "manual",
  };
}

export function useCardController(page: ComputePage): ComputeCard {
  const [edits, setEdits] = useState<ComputeDraft | null>(null);
  const autosave = page.saveMode === "autosave";
  // Autosave reads everything but the size mode from the saved settings, so a
  // card never writes back values another card has since changed. The size
  // mode is not stored, so picking Custom only lives in the card.
  const draft = autosave
    ? { ...page.base, sizeMode: edits?.sizeMode ?? page.base.sizeMode }
    : (edits ?? page.base);
  const changes = autosave ? [] : diffDraft(page.base, draft);
  const dirty = changes.length > 0;
  useReportUnsavedChanges(dirty);
  return {
    ...page,
    draft,
    changes,
    dirty,
    edit: (fn) => {
      const next = fn(draft);
      setEdits(next);
      if (autosave) {
        page.commit(next);
      }
    },
    save: () => {
      page.commit(draft);
      setEdits(null);
    },
    discard: () => setEdits(null),
  };
}
