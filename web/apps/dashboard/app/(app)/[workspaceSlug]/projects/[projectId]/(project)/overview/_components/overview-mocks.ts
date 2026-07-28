import type {
  KeyspaceStat,
  ProjectMock,
  RatelimitStat,
} from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/mock-data";
import {
  hashCode,
  mulberry32,
} from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/store";

export type ActivityEvent = {
  id: string;
  kind: "deploy" | "key" | "ratelimit" | "member" | "domain";
  text: string;
  timeAgoMin: number;
};

const ACTIVITY_ACTORS = ["dave@unkey.com", "andreas@unkey.com", "james@unkey.com", "oz@unkey.com"];

// The generator lives in the prototype folder now: the projects rail draws the
// same resources, so both surfaces have to seed their charts identically.
export { projectRequestSeries } from "@/app/(app)/[workspaceSlug]/projects/_components/prototype/series";

// 6-10 plausible events referencing the project's real apps/keyspaces/
// ratelimits, newest first. Deterministic from project.id so it's stable
// across reloads without its own localStorage-backed store.
export function activityForProject(
  project: ProjectMock,
  keyspaces: KeyspaceStat[],
  ratelimits: RatelimitStat[],
): ActivityEvent[] {
  const rand = mulberry32(hashCode(project.id));

  const shaOf = () =>
    Array.from({ length: 7 }, () => Math.floor(rand() * 16).toString(16)).join("");

  type Candidate = { kind: ActivityEvent["kind"]; text: () => string };
  const candidates: Candidate[] = [];

  for (const app of project.apps) {
    candidates.push({ kind: "deploy", text: () => `Deployed ${app.name} · ${shaOf()}` });
    candidates.push({
      kind: "domain",
      text: () => `Domain ${app.name}-${project.name}.unkey.app verified`,
    });
  }

  for (const ks of keyspaces) {
    candidates.push({ kind: "key", text: () => `Key created in ${ks.name}` });
    candidates.push({ kind: "key", text: () => `Key revoked in ${ks.name}` });
  }

  for (const rl of ratelimits) {
    candidates.push({ kind: "ratelimit", text: () => `Override added on ${rl.name}` });
    candidates.push({ kind: "ratelimit", text: () => `Limit updated on ${rl.name}` });
  }

  candidates.push({
    kind: "member",
    text: () =>
      `${ACTIVITY_ACTORS[Math.floor(rand() * ACTIVITY_ACTORS.length)]} invited a teammate`,
  });

  // Deterministic shuffle so the mix of event kinds varies per project
  // instead of always following declaration order (all deploys first, etc).
  for (let i = candidates.length - 1; i > 0; i--) {
    const j = Math.floor(rand() * (i + 1));
    [candidates[i], candidates[j]] = [candidates[j], candidates[i]];
  }

  const count = Math.min(candidates.length, 6 + Math.floor(rand() * 5));
  const chosen = candidates.slice(0, count);

  let cursor = 2 + Math.floor(rand() * 20);
  return chosen.map((candidate, i) => {
    const event: ActivityEvent = {
      id: `act_${project.id}_${i}`,
      kind: candidate.kind,
      text: candidate.text(),
      timeAgoMin: cursor,
    };
    cursor += 15 + Math.floor(rand() * 400);
    return event;
  });
}
