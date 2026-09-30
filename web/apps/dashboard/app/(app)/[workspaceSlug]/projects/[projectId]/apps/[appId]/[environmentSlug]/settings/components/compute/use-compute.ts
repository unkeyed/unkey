"use client";

import { trpc } from "@/lib/trpc/client";
import { useWorkspace } from "@/providers/workspace-provider";
import { useEffect, useMemo, useRef, useState } from "react";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateEnvironment } from "../../hooks/use-update-environment";
import { useReportUnsavedChanges } from "../../prevent-leave-context";
import {
  type AvailableRegion,
  type AvailableRegions,
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
  unschedulableIn,
} from "./model";

export type SaveMode = "autosave" | "manual";

export type ComputePage = {
  base: ComputeDraft;
  limits: ComputeLimits;
  hovered: string | null;
  setHovered: (name: string | null) => void;
  commit: (change: (current: ComputeDraft) => ComputeDraft | null) => void;
  saveMode: SaveMode;
  available: AvailableRegions;
};

export type ComputeCard = {
  view: CardView;
  limits: ComputeLimits;
  saveMode: SaveMode;
  options: AvailableRegion[];
  taken: ReadonlySet<string>;
  result: EditResult;
  blocked: string[];
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
  const regionsQuery = trpc.deploy.environmentSettings.getAvailableRegions.useQuery();
  const available: AvailableRegions = regionsQuery.data
    ? { status: "ready", regions: regionsQuery.data }
    : regionsQuery.isError
      ? { status: "error" }
      : { status: "loading" };
  return {
    available,
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

const AUTOSAVE_DELAY_MS = 600;

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
  const blocked = result.ok
    ? unschedulableIn(
        page.available,
        result.draft.regions.map((r) => r.name),
      )
    : [];
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

  const pending = useRef<CardEdit | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const flush = useRef(() => {});
  flush.current = () => {
    clearTimeout(timer.current);
    const next = pending.current;
    pending.current = null;
    if (next) {
      commitEdit(next);
    }
  };
  useEffect(() => () => flush.current(), []);

  const autosave = (next: CardEdit) => {
    const applied = applyCardEdit(page.base, slot, next);
    const names = applied.ok ? applied.draft.regions.map((r) => r.name) : [];
    if (!applied.ok || unschedulableIn(page.available, names).length > 0) {
      pending.current = null;
      clearTimeout(timer.current);
      return;
    }
    pending.current = next;
    if (next.region !== undefined) {
      flush.current();
      return;
    }
    clearTimeout(timer.current);
    timer.current = setTimeout(() => flush.current(), AUTOSAVE_DELAY_MS);
  };

  return {
    view: cardView(page.base, slot, edit),
    limits: page.limits,
    saveMode: page.saveMode,
    options: page.available.status === "ready" ? page.available.regions : [],
    taken: new Set(
      page.base.regions
        .map((r) => r.name)
        .filter((name) => slot.kind === "new" || name !== slot.name),
    ),
    result,
    blocked,
    dirty,
    edit: (patch) => {
      const next = { ...edit, ...patch };
      setEdit(next);
      if (page.saveMode === "autosave") {
        autosave(next);
      }
    },
    save: () => {
      if (dirty && blocked.length === 0) {
        commitEdit(edit);
      }
    },
  };
}
