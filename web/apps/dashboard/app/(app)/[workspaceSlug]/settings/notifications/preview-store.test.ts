import { describe, expect, it } from "vitest";
import {
  type PreviewStorage,
  createPreviewStore,
  previewStorageKey,
  previewStoreFor,
} from "./preview-store";

function memoryStorage(): PreviewStorage & { keys: () => string[] } {
  const items = new Map<string, string>();
  return {
    getItem: (key) => items.get(key) ?? null,
    setItem: (key, value) => {
      items.set(key, value);
    },
    keys: () => [...items.keys()],
  };
}

describe("createPreviewStore", () => {
  it("keeps edits, preferences, and destinations separate per workspace", () => {
    const storage = memoryStorage();
    const a = createPreviewStore("ws_a", () => storage);
    const b = createPreviewStore("ws_b", () => storage);
    const seededRules = b.getSnapshot().rules.length;

    a.actions.deleteRule(a.getSnapshot().rules[0].id);
    a.actions.deleteDestination(a.getSnapshot().destinations[0].id);
    a.actions.setEmailPreference({
      userId: "user_1",
      notificationType: "billing.paused",
      enabled: false,
    });

    expect(a.getSnapshot().rules).toHaveLength(seededRules - 1);
    expect(b.getSnapshot().rules).toHaveLength(seededRules);
    expect(b.getSnapshot().destinations).toHaveLength(3);
    expect(b.getSnapshot().preferences.some((p) => p.userId === "user_1")).toBe(false);
    expect(storage.keys()).toEqual([previewStorageKey("ws_a")]);
  });

  it("resets only its own workspace", () => {
    const storage = memoryStorage();
    const a = createPreviewStore("ws_a", () => storage);
    const b = createPreviewStore("ws_b", () => storage);
    a.actions.deleteRule(a.getSnapshot().rules[0].id);
    b.actions.deleteRule(b.getSnapshot().rules[0].id);

    a.actions.reset();

    expect(a.getSnapshot().rules).toHaveLength(4);
    expect(b.getSnapshot().rules).toHaveLength(3);
  });

  it("restores a workspace's saved state when the store is recreated", () => {
    const storage = memoryStorage();
    const first = createPreviewStore("ws_a", () => storage);
    first.actions.setRuleEnabled(first.getSnapshot().rules[0].id, false);

    const reloaded = createPreviewStore("ws_a", () => storage);
    expect(reloaded.getSnapshot().rules[0].enabled).toBe(false);
    expect(createPreviewStore("ws_b", () => storage).getSnapshot().rules[0].enabled).toBe(true);
  });

  it("returns each workspace's own state when switching away and back", () => {
    window.sessionStorage.clear();
    const first = previewStoreFor("ws_switch_a");
    first.actions.deleteRule(first.getSnapshot().rules[0].id);

    const second = previewStoreFor("ws_switch_b");
    expect(second.getSnapshot().rules).toHaveLength(4);
    second.actions.setEmailPreference({
      userId: "user_1",
      notificationType: "billing.paused",
      enabled: false,
    });
    second.actions.reset();

    const back = previewStoreFor("ws_switch_a");
    expect(back).toBe(first);
    expect(back.getSnapshot().rules).toHaveLength(3);
    expect(back.getSnapshot().preferences.some((p) => p.userId === "user_1")).toBe(false);
    expect(window.sessionStorage.getItem(previewStorageKey("ws_switch_a"))).toContain('"rules"');
  });

  it("notifies only its own subscribers", () => {
    const storage = memoryStorage();
    const a = createPreviewStore("ws_a", () => storage);
    const b = createPreviewStore("ws_b", () => storage);
    let aCalls = 0;
    let bCalls = 0;
    a.subscribe(() => aCalls++);
    b.subscribe(() => bCalls++);

    a.actions.reset();

    expect([aCalls, bCalls]).toEqual([1, 0]);
  });
});
