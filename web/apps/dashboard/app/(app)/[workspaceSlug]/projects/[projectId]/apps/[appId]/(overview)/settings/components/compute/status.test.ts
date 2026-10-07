import { describe, expect, it } from "vitest";
import { type CardEdit, type CardSlot, type ComputeDraft, applyCardEdit } from "./draft";
import { addRegionBlocker, availableFrom, cardStatus, unschedulableIn } from "./status";

const us = { name: "us-east-1", replicasMin: 1, replicasMax: 3 };
const eu = { name: "eu-central-1", replicasMin: 2, replicasMax: 5 };

const base: ComputeDraft = {
  cpuMillicores: 250,
  memoryMib: 256,
  storageMib: 0,
  regions: [us, eu],
};

describe("unschedulableIn", () => {
  it("lists the regions that cannot be scheduled once regions have loaded", () => {
    const regions = [
      { name: "us-east-1", canSchedule: true },
      { name: "eu-central-1", canSchedule: false },
    ];
    expect(
      unschedulableIn({ status: "ready", regions }, ["us-east-1", "eu-central-1", "local"]),
    ).toEqual(["eu-central-1"]);
    expect(unschedulableIn({ status: "loading" }, ["eu-central-1"])).toEqual([]);
  });
});

describe("addRegionBlocker", () => {
  const regions = [
    { name: "us-east-1", canSchedule: true },
    { name: "eu-central-1", canSchedule: false },
    { name: "us-west-2", canSchedule: true },
  ];

  it("explains why no region can be added", () => {
    expect(addRegionBlocker({ status: "loading" }, ["us-east-1"])).toBe("Loading regions…");
    expect(addRegionBlocker({ status: "error" }, ["us-east-1"])).toBe(
      "Couldn't load regions. Reload the page to try again.",
    );
    expect(addRegionBlocker({ status: "ready", regions }, ["us-east-1", "us-west-2"])).toBe(
      "Your app already runs in every available region.",
    );
  });

  it("allows adding while a schedulable region is free", () => {
    expect(addRegionBlocker({ status: "ready", regions }, ["us-east-1"])).toBeNull();
  });
});

describe("cardStatus", () => {
  const ready = {
    status: "ready" as const,
    regions: [
      { name: "us-east-1", canSchedule: false },
      { name: "eu-central-1", canSchedule: true },
      { name: "us-west-2", canSchedule: true },
      { name: "ap-southeast-1", canSchedule: false },
    ],
  };
  const usCard: CardSlot = { kind: "saved", name: "us-east-1" };
  const status = (slot: CardSlot, edit: CardEdit) =>
    cardStatus(applyCardEdit(base, slot, edit), ready, base);

  it("lets a size edit save while an existing region is unschedulable", () => {
    expect(status(usCard, { storageMib: 512 })).toEqual({
      type: "dirty",
      draft: { ...base, storageMib: 512 },
    });
  });

  it("blocks only regions the edit adds", () => {
    expect(status({ kind: "new" }, { region: "ap-southeast-1" })).toEqual({
      type: "blocked",
      names: ["ap-southeast-1"],
    });
    expect(status(usCard, { region: "us-west-2" }).type).toBe("dirty");
  });

  it("reports taken, unpicked, missing and clean", () => {
    expect(status(usCard, { region: "eu-central-1" })).toEqual({ type: "taken" });
    expect(status({ kind: "new" }, {})).toEqual({ type: "invalid", reason: "unpicked" });
    expect(status({ kind: "saved", name: "us-west-2" }, { storageMib: 512 })).toEqual({
      type: "invalid",
      reason: "missing",
    });
    expect(status(usCard, { storageMib: 0 })).toEqual({ type: "clean" });
  });
});

describe("availableFrom", () => {
  it("prefers loaded regions over an error and waits otherwise", () => {
    const regions = [{ name: "us-east-1", canSchedule: true }];
    expect(availableFrom({ data: regions, isError: true })).toEqual({ status: "ready", regions });
    expect(availableFrom({ data: undefined, isError: true })).toEqual({ status: "error" });
    expect(availableFrom({ data: undefined, isError: false })).toEqual({ status: "loading" });
  });
});
