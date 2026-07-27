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

// Hourly {valid, error} pairs shaped like real traffic: a business-hours hump,
// per-point noise, rare short spikes, and errors as a small share of valid
// traffic with occasional blips. `points[i]` is `points - 1 - i` hours ago.
export function projectRequestSeries(
  seedKey: string,
  points: number,
  baseMagnitude: number,
): { valid: number; error: number }[] {
  const rand = mulberry32(hashCode(seedKey));
  const series: { valid: number; error: number }[] = [];

  const diurnalMultiplier = (i: number) => {
    const hourOfDay = (points - 1 - i) % 24;
    const phase = ((hourOfDay - 14) / 24) * Math.PI * 2;
    return 0.35 + 0.65 * (0.5 + 0.5 * Math.cos(phase));
  };

  let spikeCooldown = 0;

  for (let i = 0; i < points; i++) {
    const noise = 0.85 + rand() * 0.3;
    let magnitude = baseMagnitude * diurnalMultiplier(i) * noise;

    if (spikeCooldown > 0) {
      spikeCooldown--;
    } else if (rand() < 0.03) {
      magnitude *= 1.8 + rand() * 1.4;
      spikeCooldown = 1 + Math.floor(rand() * 2);
    }

    const valid = Math.max(0, Math.round(magnitude));
    const errorBlip = rand() < 0.05;
    const errorRate = errorBlip ? 0.04 + rand() * 0.08 : 0.005 + rand() * 0.035;
    const error = Math.max(0, Math.round(valid * errorRate));

    series.push({ valid, error });
  }

  return series;
}

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
