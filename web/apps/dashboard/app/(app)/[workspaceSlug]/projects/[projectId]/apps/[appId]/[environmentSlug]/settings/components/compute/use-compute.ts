"use client";

import { useWorkspace } from "@/providers/workspace-provider";
import { useMemo, useState } from "react";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateEnvironment } from "../../hooks/use-update-environment";
import { useReportUnsavedChanges } from "../../prevent-leave-context";
import {
  type CardEdit,
  type CardSlot,
  type CardView,
  type ComputeDraft,
  type ComputeLimits,
  type EditResult,
  applyCardEdit,
  applyDraft,
  cardView,
  fromSettings,
  resolveLimits,
  sameDraft,
} from "./model";

export type SaveMode = "autosave" | "manual";

export type ComputePage = {
  base: ComputeDraft;
  limits: ComputeLimits;
  hovered: string | null;
  setHovered: (name: string | null) => void;
  commit: (change: (current: ComputeDraft) => ComputeDraft | null) => void;
  saveMode: SaveMode;
};

export type ComputeCard = {
  view: CardView;
  limits: ComputeLimits;
  saveMode: SaveMode;
  taken: ReadonlySet<string>;
  result: EditResult;
  dirty: boolean;
  edit: (patch: CardEdit) => void;
  save: () => void;
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
    commit: (change) =>
      updateEnvironment((target) => {
        const next = change(fromSettings(target));
        if (next) {
          applyDraft(target, next);
        }
      }),
    saveMode: variant === "onboarding" ? "autosave" : "manual",
  };
}

function modesOf({ sizeMode, storageMode }: CardEdit): CardEdit {
  return { sizeMode, storageMode };
}

export function useCardController(
  page: ComputePage,
  slot: CardSlot,
  onSaved: (region: string) => void,
): ComputeCard {
  const [edit, setEdit] = useState<CardEdit>({});
  const result = applyCardEdit(page.base, slot, edit);
  const dirty = result.ok && !sameDraft(result.draft, page.base);
  useReportUnsavedChanges(page.saveMode === "manual" && dirty);

  const commitEdit = (next: CardEdit) => {
    if (next.region !== undefined) {
      onSaved(next.region);
    }
    page.commit((current) => {
      const applied = applyCardEdit(current, slot, next);
      return applied.ok ? applied.draft : null;
    });
    setEdit(modesOf(next));
  };

  return {
    view: cardView(page.base, slot, edit),
    limits: page.limits,
    saveMode: page.saveMode,
    taken: new Set(
      page.base.regions
        .map((r) => r.name)
        .filter((name) => slot.kind === "new" || name !== slot.name),
    ),
    result,
    dirty,
    edit: (patch) => {
      const next = { ...edit, ...patch };
      setEdit(next);
      if (page.saveMode === "autosave" && applyCardEdit(page.base, slot, next).ok) {
        commitEdit(next);
      }
    },
    save: () => {
      if (dirty) {
        commitEdit(edit);
      }
    },
  };
}
