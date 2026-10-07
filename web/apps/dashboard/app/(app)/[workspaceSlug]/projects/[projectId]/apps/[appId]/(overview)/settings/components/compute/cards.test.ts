import { describe, expect, it } from "vitest";
import {
  beginSave,
  computeList,
  editCard,
  editOf,
  finishSave,
  footerNote,
  initialUi,
  openAdd,
  resetCard,
  toggleCard,
} from "./cards";
import type { CardSlot, ComputeDraft } from "./draft";

const us = { name: "us-east-1", replicasMin: 1, replicasMax: 3 };
const eu = { name: "eu-central-1", replicasMin: 2, replicasMax: 5 };

const base: ComputeDraft = {
  cpuMillicores: 250,
  memoryMib: 256,
  storageMib: 0,
  regions: [us, eu],
};

describe("footerNote", () => {
  it("names the reason a card cannot save", () => {
    expect(footerNote({ type: "taken" }, "manual")).toEqual({
      tone: "error",
      text: "Your app already runs in this region. Pick another one.",
    });
    expect(footerNote({ type: "blocked", names: ["us-east-1", "local"] }, "manual")).toEqual({
      tone: "warning",
      text: "Can't add us-east-1, local right now. Pick another region to save.",
    });
    expect(footerNote({ type: "invalid", reason: "missing" }, "autosave")).toEqual({
      tone: "error",
      text: "This region is no longer in this environment. Reload to edit it.",
    });
    expect(footerNote({ type: "invalid", reason: "unpicked" }, "manual")).toBeNull();
  });

  it("notes pending changes only when the card saves manually", () => {
    expect(footerNote({ type: "dirty", draft: base }, "manual")).toEqual({
      tone: "muted",
      text: "Changes apply to the next deployment",
    });
    expect(footerNote({ type: "dirty", draft: base }, "autosave")).toBeNull();
    expect(footerNote({ type: "clean" }, "manual")).toBeNull();
  });
});

const usCard: CardSlot = { kind: "saved", name: "us-east-1" };
const newCard: CardSlot = { kind: "new" };
const names = ["us-east-1", "eu-central-1"];
const ready = {
  status: "ready" as const,
  regions: [
    { name: "us-east-1", canSchedule: true },
    { name: "eu-central-1", canSchedule: true },
    { name: "us-west-2", canSchedule: true },
    { name: "ap-southeast-1", canSchedule: false },
  ],
};

describe("initialUi", () => {
  it("opens the first card only when the section autosaves", () => {
    expect(initialUi(names, "autosave").cards).toEqual({
      "us-east-1": { open: true, pick: null, saving: false },
    });
    expect(initialUi(names, "manual").cards).toEqual({});
  });
});

describe("editCard", () => {
  it("shares size edits with every card and keeps the region pick on its own card", () => {
    const ui = editCard(initialUi(names, "manual"), usCard, {
      region: "us-west-2",
      storageMib: 512,
    });
    expect(editOf(ui, usCard)).toEqual({ storageMib: 512, region: "us-west-2" });
    expect(editOf(ui, { kind: "saved", name: "eu-central-1" })).toEqual({ storageMib: 512 });
  });

  it("resets a card to its saved values but keeps the size and storage modes", () => {
    const edited = editCard(initialUi(names, "manual"), usCard, {
      region: "us-west-2",
      sizeMode: "custom",
      storageMib: 512,
    });
    const reset = resetCard(edited, usCard, editOf(edited, usCard));
    expect(editOf(reset, usCard)).toEqual({ sizeMode: "custom", storageMode: undefined });
  });
});

describe("beginSave and finishSave", () => {
  const opened = toggleCard(initialUi(names, "manual"), "us-east-1");
  const swap = { region: "us-west-2" };

  it("keeps a renamed card open under its new name and clears the pick once saved", () => {
    const picked = editCard(opened, usCard, swap);
    const saving = beginSave(picked, usCard, swap);
    expect(saving.cards).toEqual({ "us-west-2": { open: true, pick: "us-west-2", saving: true } });
    expect(finishSave(saving, usCard, swap, true).cards).toEqual({
      "us-west-2": { open: true, pick: null, saving: false },
    });
  });

  it("moves a failed rename back to the saved name and keeps the pick", () => {
    const saving = beginSave(editCard(opened, usCard, swap), usCard, swap);
    expect(finishSave(saving, usCard, swap, false).cards).toEqual({
      "us-east-1": { open: true, pick: "us-west-2", saving: false },
    });
  });

  it("drops the shared edit after a save unless it changed during the save", () => {
    const edit = { storageMode: "preset" as const, storageMib: 512 };
    const saving = beginSave(editCard(opened, usCard, edit), usCard, edit);
    expect(finishSave(saving, usCard, edit, true).shared).toEqual({
      sizeMode: undefined,
      storageMode: "preset",
    });
    const changed = editCard(saving, usCard, { storageMib: 1024 });
    expect(finishSave(changed, usCard, edit, true).shared).toEqual({ storageMib: 1024 });
  });

  it("turns a saved new region into an open card that pops in", () => {
    const add = { region: "us-west-2" };
    const adding = editCard(openAdd(initialUi(names, "manual")), newCard, add);
    const saving = beginSave(adding, newCard, add);
    expect(saving.add).toEqual({ type: "open", pick: "us-west-2", saving: "us-west-2" });
    const saved = finishSave(saving, newCard, add, true);
    expect([saved.add, saved.popIn, saved.cards["us-west-2"]]).toEqual([
      { type: "closed" },
      "us-west-2",
      { open: true, pick: null, saving: false },
    ]);
    expect(finishSave(saving, newCard, add, false).add).toEqual({
      type: "open",
      pick: "us-west-2",
      saving: null,
    });
  });
});

describe("computeList", () => {
  const list = (ui = initialUi(names, "manual"), saveMode: "manual" | "autosave" = "manual") =>
    computeList({ base, ui, available: ready, saveMode, hovered: "eu-central-1" });

  it("renders a closed card per saved region and offers to add one", () => {
    const { cards, addRegion, dirty } = list();
    expect(cards.map((c) => c.frame)).toEqual([
      {
        type: "saved",
        name: "us-east-1",
        shown: "us-east-1",
        unavailable: false,
        summary: "Mixed instances · XS · 1/4 vCPU · 256 MiB",
        open: false,
        highlighted: false,
        popIn: false,
      },
      {
        type: "saved",
        name: "eu-central-1",
        shown: "eu-central-1",
        unavailable: false,
        summary: "Mixed instances · XS · 1/4 vCPU · 256 MiB",
        open: false,
        highlighted: true,
        popIn: false,
      },
    ]);
    expect(cards[0]?.footer).toEqual({
      leading: { type: "remove", name: "us-east-1" },
      note: null,
      save: { type: "disabled" },
    });
    expect([addRegion, dirty]).toEqual([{ type: "available" }, false]);
  });

  it("marks every card dirty for a shared edit and disables taken regions", () => {
    const ui = toggleCard(
      editCard(initialUi(names, "manual"), usCard, { storageMib: 512 }),
      "us-east-1",
    );
    const { cards, dirty } = list(ui);
    expect(cards.map((c) => c.footer.save)).toEqual([{ type: "ready" }, { type: "ready" }]);
    expect(cards[0]?.frame).toMatchObject({
      open: true,
      summary: "Same settings in every region for now",
    });
    expect(cards[0]?.options).toEqual([
      { name: "us-east-1", canSchedule: true, disabled: false },
      { name: "eu-central-1", canSchedule: true, disabled: true },
      { name: "us-west-2", canSchedule: true, disabled: false },
      { name: "ap-southeast-1", canSchedule: false, disabled: true },
    ]);
    expect(dirty).toBe(true);
  });

  it("shows the new card last and hides the saved card for a region being added", () => {
    const add = { region: "us-west-2" };
    const adding = editCard(openAdd(initialUi(names, "manual")), newCard, add);
    const idle = list(adding);
    expect(idle.cards.map((c) => c.key)).toEqual(["us-east-1", "eu-central-1", "new-region"]);
    expect(idle.cards[2]?.footer).toEqual({
      leading: { type: "cancel" },
      note: { tone: "muted", text: "Changes apply to the next deployment" },
      save: { type: "ready" },
    });
    expect(idle.addRegion).toEqual({ type: "adding" });

    const withAdded = {
      ...base,
      regions: [...base.regions, { name: "us-west-2", replicasMin: 1, replicasMax: 3 }],
    };
    const saving = computeList({
      base: withAdded,
      ui: beginSave(adding, newCard, add),
      available: ready,
      saveMode: "manual",
      hovered: null,
    });
    expect(saving.cards.map((c) => c.key)).toEqual(["us-east-1", "eu-central-1", "new-region"]);
    expect(saving.cards[2]?.footer.save).toEqual({ type: "saving" });
  });

  it("hides the save button and the pending note when the section autosaves", () => {
    const ui = editCard(initialUi(names, "autosave"), usCard, { storageMib: 512 });
    expect(list(ui, "autosave").cards[0]?.footer).toEqual({
      leading: { type: "remove", name: "us-east-1" },
      note: null,
      save: { type: "none" },
    });
  });
});
