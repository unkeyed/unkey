import { NEXT_DEPLOY } from "@/lib/collections/deploy/pending-redeploy";
import { match } from "@unkey/match";
import {
  type CardEdit,
  type CardSlot,
  type CardView,
  type ComputeDraft,
  type SharedEdit,
  applyCardEdit,
  cardView,
  modesOf,
  sameSharedEdit,
  sharedPart,
} from "./draft";
import { inheritedSummary } from "./format";
import {
  type AvailableRegions,
  type CardStatus,
  addRegionBlocker,
  cardStatus,
  unschedulableIn,
} from "./status";

const SHARED_SCOPE_NOTE = "Same settings in every region for now";

export type SaveMode = "autosave" | "manual";

type CardUi = { open: boolean; pick: string | null; saving: boolean };

type AddCard = { type: "closed" } | { type: "open"; pick: string | null; saving: string | null };

/** Size, storage and replicas apply to every region, so every card edits the one `shared` copy. */
type ComputeUi = {
  shared: SharedEdit;
  cards: Readonly<Record<string, CardUi>>;
  add: AddCard;
  popIn: string | null;
};

const IDLE: CardUi = { open: false, pick: null, saving: false };

export function initialUi(names: string[], saveMode: SaveMode): ComputeUi {
  const first = names.at(0);
  return {
    shared: {},
    cards: saveMode === "autosave" && first ? { [first]: { ...IDLE, open: true } } : {},
    add: { type: "closed" },
    popIn: null,
  };
}

function cardUi(ui: ComputeUi, name: string): CardUi {
  return ui.cards[name] ?? IDLE;
}

function pickOf(ui: ComputeUi, slot: CardSlot): string | null {
  if (slot.kind === "saved") {
    return cardUi(ui, slot.name).pick;
  }
  return ui.add.type === "open" ? ui.add.pick : null;
}

export function editOf(ui: ComputeUi, slot: CardSlot): CardEdit {
  const pick = pickOf(ui, slot);
  return pick === null ? ui.shared : { ...ui.shared, region: pick };
}

function withCard(ui: ComputeUi, name: string, card: CardUi): ComputeUi {
  return { ...ui, cards: { ...ui.cards, [name]: card } };
}

function moveCard(ui: ComputeUi, from: string, to: string, card: CardUi): ComputeUi {
  const { [from]: _, ...rest } = ui.cards;
  return { ...ui, cards: { ...rest, [to]: card } };
}

function withPick(ui: ComputeUi, slot: CardSlot, pick: string | null): ComputeUi {
  if (slot.kind === "saved") {
    return withCard(ui, slot.name, { ...cardUi(ui, slot.name), pick });
  }
  return ui.add.type === "open" ? { ...ui, add: { ...ui.add, pick } } : ui;
}

export function toggleCard(ui: ComputeUi, name: string): ComputeUi {
  const card = cardUi(ui, name);
  return withCard(ui, name, { ...card, open: !card.open });
}

export function openAdd(ui: ComputeUi): ComputeUi {
  return { ...ui, add: { type: "open", pick: null, saving: null } };
}

export function closeAdd(ui: ComputeUi): ComputeUi {
  return { ...ui, add: { type: "closed" } };
}

export function editCard(ui: ComputeUi, slot: CardSlot, next: CardEdit): ComputeUi {
  const shared = sharedPart(next);
  const picked = withPick(ui, slot, next.region ?? null);
  return sameSharedEdit(shared, ui.shared) ? picked : { ...picked, shared };
}

export function resetCard(ui: ComputeUi, slot: CardSlot, next: CardEdit): ComputeUi {
  return { ...withPick(ui, slot, null), shared: modesOf(next) };
}

/** Moves a saved card's state to the region it is renaming to, so the card stays open while the optimistic write lands. */
export function beginSave(ui: ComputeUi, slot: CardSlot, edit: CardEdit): ComputeUi {
  if (slot.kind === "saved") {
    const to = edit.region ?? slot.name;
    return moveCard(ui, slot.name, to, { ...cardUi(ui, slot.name), saving: true });
  }
  if (ui.add.type === "closed" || edit.region === undefined) {
    return ui;
  }
  return { ...ui, add: { ...ui.add, saving: edit.region } };
}

export function finishSave(
  ui: ComputeUi,
  slot: CardSlot,
  edit: CardEdit,
  persisted: boolean,
): ComputeUi {
  const shared =
    persisted && sameSharedEdit(ui.shared, sharedPart(edit)) ? modesOf(edit) : ui.shared;
  if (slot.kind === "saved") {
    const to = edit.region ?? slot.name;
    const card = { ...cardUi(ui, to), saving: false };
    if (!persisted) {
      return moveCard(ui, to, slot.name, card);
    }
    const pick = card.pick === (edit.region ?? null) ? null : card.pick;
    return { ...withCard(ui, to, { ...card, pick }), shared };
  }
  if (edit.region === undefined) {
    return ui;
  }
  if (!persisted) {
    return ui.add.type === "open" ? { ...ui, add: { ...ui.add, saving: null } } : ui;
  }
  return {
    ...withCard(ui, edit.region, { ...IDLE, open: true }),
    shared,
    add: { type: "closed" },
    popIn: edit.region,
  };
}

type RegionOption = { name: string; canSchedule: boolean; disabled: boolean };

type CardFrame =
  | {
      type: "saved";
      name: string;
      shown: string;
      unavailable: boolean;
      summary: string;
      open: boolean;
      highlighted: boolean;
      popIn: boolean;
    }
  | { type: "new"; note: string | null };

export type FooterLeading =
  | { type: "remove"; name: string }
  | { type: "keep-one" }
  | { type: "cancel" };

export type FooterNote = { tone: "error" | "warning" | "muted"; text: string };

export type SaveButton =
  | { type: "none" }
  | { type: "disabled" }
  | { type: "ready" }
  | { type: "saving" };

type CardFooter = { leading: FooterLeading; note: FooterNote | null; save: SaveButton };

export type RegionCardState = {
  key: string;
  slot: CardSlot;
  frame: CardFrame;
  view: CardView;
  options: RegionOption[];
  footer: CardFooter;
};

export type AddRegionState =
  | { type: "adding" }
  | { type: "available" }
  | { type: "blocked"; reason: string };

type ComputeList = {
  cards: RegionCardState[];
  addRegion: AddRegionState;
  unavailable: string[];
  dirty: boolean;
};

type ListInput = {
  base: ComputeDraft;
  ui: ComputeUi;
  available: AvailableRegions;
  saveMode: SaveMode;
  hovered: string | null;
};

export function statusOf(
  { base, available }: Pick<ListInput, "base" | "available">,
  slot: CardSlot,
  edit: CardEdit,
): CardStatus {
  return cardStatus(applyCardEdit(base, slot, edit), available, base);
}

export function footerNote(status: CardStatus, mode: SaveMode): FooterNote | null {
  return match(status)
    .with({ type: "invalid" }, ({ reason }) =>
      reason === "missing"
        ? {
            tone: "error" as const,
            text: "This region is no longer in this environment. Reload to edit it.",
          }
        : null,
    )
    .with({ type: "taken" }, () => ({
      tone: "error" as const,
      text: "Your app already runs in this region. Pick another one.",
    }))
    .with({ type: "blocked" }, ({ names }) => ({
      tone: "warning" as const,
      text: `Can't add ${names.join(", ")} right now. Pick another region to save.`,
    }))
    .with({ type: "dirty" }, () =>
      mode === "manual" ? { tone: "muted" as const, text: NEXT_DEPLOY } : null,
    )
    .with({ type: "clean" }, () => null)
    .exhaustive();
}

function saveButton(status: CardStatus, saving: boolean, mode: SaveMode): SaveButton {
  if (mode === "autosave") {
    return { type: "none" };
  }
  if (saving) {
    return { type: "saving" };
  }
  return status.type === "dirty" ? { type: "ready" } : { type: "disabled" };
}

function cardFooter(
  leading: FooterLeading,
  status: CardStatus,
  saving: boolean,
  mode: SaveMode,
): CardFooter {
  return {
    leading,
    note: saving ? { tone: "muted", text: "Saving…" } : footerNote(status, mode),
    save: saveButton(status, saving, mode),
  };
}

function regionOptions(
  { base, available }: ListInput,
  slot: CardSlot,
  region: string | null,
): RegionOption[] {
  if (available.status !== "ready") {
    return [];
  }
  const own = slot.kind === "saved" ? slot.name : null;
  const taken = new Set(base.regions.map((r) => r.name).filter((name) => name !== own));
  return available.regions.map(({ name, canSchedule }) => ({
    name,
    canSchedule,
    disabled: !canSchedule || (name !== region && taken.has(name)),
  }));
}

function savedCard(input: ListInput, name: string): { card: RegionCardState; status: CardStatus } {
  const { base, ui, available, saveMode, hovered } = input;
  const slot: CardSlot = { kind: "saved", name };
  const edit = editOf(ui, slot);
  const view = cardView(base, slot, edit);
  const status = statusOf(input, slot, edit);
  const { open, saving } = cardUi(ui, name);
  const shown = view.region ?? name;
  const shared = base.regions.length > 1;
  const card: RegionCardState = {
    key: name,
    slot,
    frame: {
      type: "saved",
      name,
      shown,
      unavailable: unschedulableIn(available, [shown]).length > 0,
      summary: shared && open ? SHARED_SCOPE_NOTE : inheritedSummary(view),
      open,
      highlighted: hovered === name,
      popIn: ui.popIn === name,
    },
    view,
    options: regionOptions(input, slot, view.region),
    footer: cardFooter(
      shared ? { type: "remove", name } : { type: "keep-one" },
      status,
      saving,
      saveMode,
    ),
  };
  return { card, status };
}

function newCard(input: ListInput, saving: boolean): { card: RegionCardState; status: CardStatus } {
  const { base, ui, saveMode } = input;
  const slot: CardSlot = { kind: "new" };
  const edit = editOf(ui, slot);
  const view = cardView(base, slot, edit);
  const status = statusOf(input, slot, edit);
  const card: RegionCardState = {
    key: "new-region",
    slot,
    frame: { type: "new", note: base.regions.length > 0 ? SHARED_SCOPE_NOTE : null },
    view,
    options: regionOptions(input, slot, view.region),
    footer: cardFooter({ type: "cancel" }, status, saving, saveMode),
  };
  return { card, status };
}

function addRegionState(
  adding: boolean,
  available: AvailableRegions,
  names: string[],
): AddRegionState {
  if (adding) {
    return { type: "adding" };
  }
  const reason = addRegionBlocker(available, names);
  return reason === null ? { type: "available" } : { type: "blocked", reason };
}

export function computeList(input: ListInput): ComputeList {
  const { base, ui, available } = input;
  const names = base.regions.map((r) => r.name);
  const adding = ui.add.type === "open" ? ui.add : null;
  const entries = names
    .filter((name) => name !== adding?.saving)
    .map((name) => savedCard(input, name));
  if (adding) {
    entries.push(newCard(input, adding.saving !== null));
  }
  return {
    cards: entries.map((e) => e.card),
    addRegion: addRegionState(adding !== null, available, names),
    unavailable: unschedulableIn(available, names),
    dirty: entries.some((e) => e.status.type === "dirty"),
  };
}
