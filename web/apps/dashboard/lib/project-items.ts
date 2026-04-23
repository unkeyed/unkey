/**
 * Prototype-only: localStorage-backed contents of a project.
 *
 * Andreas's reframe on 2026-04-21 is that a project is a container — it
 * holds one or more apps plus supplementary services (databases, queues,
 * vault). The DB already models this; the UI hasn't exposed it yet.
 *
 * This lib fakes it with localStorage so we can feel the sidebar + nav
 * shape before the backend is ready. When a real items API arrives, the
 * hook/lib get swapped out and consumers don't change.
 */

export type ProjectItemType = "app" | "database" | "queue" | "vault";

export type ProjectItem = {
  id: string;
  type: ProjectItemType;
  name: string;
  slug: string;
};

export const PROJECT_ITEMS_EVENT = "unkey:project-items-changed";
export const PROJECT_ITEMS_TYPE_ORDER: ProjectItemType[] = ["app", "database", "queue", "vault"];

export function storageKey(projectId: string): string {
  return `unkey:project-items:${projectId}`;
}

function slugify(name: string): string {
  return name
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

function randomId(): string {
  return `itm_${Math.random().toString(36).slice(2, 10)}`;
}

function seed(): ProjectItem[] {
  return [{ id: randomId(), type: "app", name: "local api", slug: "local-api" }];
}

function read(projectId: string): ProjectItem[] | null {
  if (typeof window === "undefined") {
    return null;
  }
  const raw = window.localStorage.getItem(storageKey(projectId));
  if (!raw) {
    return null;
  }
  try {
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) {
      return null;
    }
    return parsed.filter(isProjectItem);
  } catch {
    return null;
  }
}

function isProjectItem(value: unknown): value is ProjectItem {
  if (!value || typeof value !== "object") {
    return false;
  }
  const v = value as Record<string, unknown>;
  return (
    typeof v.id === "string" &&
    typeof v.name === "string" &&
    typeof v.slug === "string" &&
    (v.type === "app" || v.type === "database" || v.type === "queue" || v.type === "vault")
  );
}

export function getProjectItems(projectId: string): ProjectItem[] {
  const existing = read(projectId);
  if (existing) {
    return existing;
  }
  const seeded = seed();
  setProjectItems(projectId, seeded);
  return seeded;
}

export function setProjectItems(projectId: string, items: ProjectItem[]): void {
  if (typeof window === "undefined") {
    return;
  }
  window.localStorage.setItem(storageKey(projectId), JSON.stringify(items));
  window.dispatchEvent(new Event(PROJECT_ITEMS_EVENT));
}

export function addProjectItem(
  projectId: string,
  input: { type: ProjectItemType; name: string },
): ProjectItem {
  const current = getProjectItems(projectId);
  const baseSlug = slugify(input.name) || input.type;
  const slug = uniqueSlug(baseSlug, current);
  const item: ProjectItem = {
    id: randomId(),
    type: input.type,
    name: input.name.trim(),
    slug,
  };
  setProjectItems(projectId, [...current, item]);
  return item;
}

function uniqueSlug(base: string, items: ProjectItem[]): string {
  const taken = new Set(items.map((i) => i.slug));
  if (!taken.has(base)) {
    return base;
  }
  let n = 2;
  while (taken.has(`${base}-${n}`)) {
    n++;
  }
  return `${base}-${n}`;
}

export function sortByTypeGroup(items: ProjectItem[]): ProjectItem[] {
  const rank = new Map(PROJECT_ITEMS_TYPE_ORDER.map((t, i) => [t, i]));
  return [...items].sort((a, b) => {
    const diff = (rank.get(a.type) ?? 0) - (rank.get(b.type) ?? 0);
    if (diff !== 0) {
      return diff;
    }
    return a.name.localeCompare(b.name);
  });
}
