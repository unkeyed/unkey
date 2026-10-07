"use client";

import { collection } from "@/lib/collections";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { resolveLimits } from "@/lib/compute/sizing";
import { trpc } from "@/lib/trpc/client";
import { useWorkspace } from "@/providers/workspace-provider";
import { useReportUnsavedChanges } from "@unkey/ui";
import { useEffect, useMemo, useRef, useState } from "react";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateAllEnvironments } from "../../hooks/use-update-all-environments";
import {
  type RegionCardState,
  type SaveMode,
  beginSave,
  closeAdd,
  computeList,
  editCard,
  editOf,
  finishSave,
  initialUi,
  openAdd,
  resetCard,
  statusOf,
  toggleCard,
} from "./cards";
import {
  type CardEdit,
  type CardSlot,
  type ComputeDraft,
  applyCardEdit,
  applyDraft,
  fromSettings,
  removeRegion,
} from "./draft";
import { availableFrom } from "./status";

const AUTOSAVE_DELAY_MS = 600;

export type ComputeActions = ReturnType<typeof useCompute>["actions"];

export function useCompute() {
  const { settings, autoSave } = useEnvironmentSettings();
  const { limits } = useWorkspace();
  const saveMode: SaveMode = autoSave ? "autosave" : "manual";
  const base = useMemo(() => fromSettings(settings), [settings]);
  const regionsQuery = trpc.deploy.environmentSettings.getAvailableRegions.useQuery();
  const available = availableFrom(regionsQuery);
  const [ui, setUi] = useState(() =>
    initialUi(
      base.regions.map((r) => r.name),
      saveMode,
    ),
  );
  const [hovered, setHovered] = useState<string | null>(null);
  const list = computeList({ base, ui, available, saveMode, hovered });
  useReportUnsavedChanges(saveMode === "manual" && list.dirty);

  const commit = useCommit(autoSave, settings.environmentId);
  const commitCard = async (slot: CardSlot, edit: CardEdit) => {
    setUi((current) => beginSave(current, slot, edit));
    const persisted = await commit((current) => {
      const applied = applyCardEdit(current, slot, edit);
      return applied.ok ? applied.draft : null;
    });
    setUi((current) => finishSave(current, slot, edit, persisted));
  };
  const autosave = useAutosave(commitCard);

  const edit = (slot: CardSlot, patch: CardEdit) => {
    const next = { ...editOf(ui, slot), ...patch };
    setUi((current) => editCard(current, slot, next));
    if (saveMode === "manual") {
      return;
    }
    const status = statusOf({ base, available }, slot, next);
    if (status.type === "dirty") {
      autosave.schedule(slot, next, next.region !== undefined);
      return;
    }
    autosave.cancel();
    if (status.type === "clean") {
      setUi((current) => resetCard(current, slot, next));
    }
  };

  return {
    draft: base,
    limits: resolveLimits(limits),
    list,
    hovered,
    actions: {
      hover: setHovered,
      toggle: (name: string) => setUi((current) => toggleCard(current, name)),
      startAdd: () => setUi(openAdd),
      cancelAdd: () => setUi(closeAdd),
      edit,
      save: (card: RegionCardState) => {
        if (card.footer.save.type === "ready") {
          void commitCard(card.slot, editOf(ui, card.slot));
        }
      },
      remove: (name: string) => commit((current) => removeRegion(current, name)),
    },
  };
}

function useCommit(autoSave: boolean, environmentId: string) {
  const updateAllEnvironments = useUpdateAllEnvironments();
  return async (change: (current: ComputeDraft) => ComputeDraft | null): Promise<boolean> => {
    const updater = (target: EnvironmentSettings) => {
      const next = change(fromSettings(target));
      if (next) {
        applyDraft(target, next);
      }
    };
    // Only onboarding autosaves, and it has no environment scope yet, so its one form
    // seeds every environment.
    const tx = autoSave
      ? updateAllEnvironments(updater)
      : collection.environmentSettings.update(environmentId, updater);
    if (!tx) {
      return false;
    }
    try {
      await tx.isPersisted.promise;
      return true;
    } catch {
      return false;
    }
  };
}

function useAutosave(commitCard: (slot: CardSlot, edit: CardEdit) => Promise<void>) {
  const pending = useRef<{ slot: CardSlot; edit: CardEdit } | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const flush = useRef(() => {});
  flush.current = () => {
    clearTimeout(timer.current);
    const next = pending.current;
    pending.current = null;
    if (next) {
      void commitCard(next.slot, next.edit);
    }
  };
  useEffect(() => () => flush.current(), []);
  return {
    schedule: (slot: CardSlot, edit: CardEdit, immediate: boolean) => {
      pending.current = { slot, edit };
      if (immediate) {
        flush.current();
        return;
      }
      clearTimeout(timer.current);
      timer.current = setTimeout(() => flush.current(), AUTOSAVE_DELAY_MS);
    },
    cancel: () => {
      pending.current = null;
      clearTimeout(timer.current);
    },
  };
}
