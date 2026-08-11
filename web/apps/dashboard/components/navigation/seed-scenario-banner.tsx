"use client";

import type { Route } from "next";
import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useRef, useState } from "react";

/**
 * Local review scaffolding, not for production. Says which billing scenario is
 * seeded, and applies one when the URL carries `?scenario=`, so the states can
 * be flicked through from the address bar.
 *
 * Colours are inline rather than Tailwind classes: a class name assembled from
 * data at runtime is not in the stylesheet Tailwind generated.
 */
const COLOURS: Record<string, string> = {
  "no-plan": "#667085",
  "starter-over": "#d92d20",
  "pro-healthy": "#079455",
  "business-high": "#7839ee",
  suspended: "#b42318",
  "budget-no-stop": "#dc6803",
  "budget-stop": "#b54708",
  "api-over-quota": "#e04f16",
  "both-over": "#912018",
  "zero-usage": "#475467",
  "under-credit": "#0e9384",
  unattributed: "#4e5ba6",
};

const FALLBACK = "#344054";
const PENDING = "#344054";

async function readActive(): Promise<string | null> {
  const res = await fetch("/api/dev/seed-scenario", { cache: "no-store" });
  if (!res.ok) {
    return null;
  }
  const body: { scenario: string | null } = await res.json();
  return body.scenario;
}

export function SeedScenarioBanner() {
  const router = useRouter();
  const params = useSearchParams();
  const requested = params.get("scenario");

  const [active, setActive] = useState<string | null>(null);
  const [applying, setApplying] = useState<string | null>(null);
  const [failed, setFailed] = useState<string | null>(null);
  const [loaded, setLoaded] = useState(false);
  const handled = useRef<Set<string>>(new Set());

  useEffect(() => {
    let cancelled = false;
    const sync = () => {
      readActive()
        .then((scenario) => {
          if (!cancelled) {
            setActive(scenario);
            setLoaded(true);
          }
        })
        .catch(() => {
          if (!cancelled) {
            setLoaded(true);
          }
        });
    };

    sync();
    // Re-read on focus so switching in a terminal shows up on return.
    window.addEventListener("focus", sync);
    return () => {
      cancelled = true;
      window.removeEventListener("focus", sync);
    };
  }, []);

  useEffect(() => {
    // Wait for the marker: applying before knowing what is already seeded would
    // re-run the same scenario on every reload.
    if (!loaded || !requested || handled.current.has(requested)) {
      return;
    }
    handled.current.add(requested);
    // Already seeded, so the param is just describing the current state.
    if (requested === active) {
      return;
    }

    let cancelled = false;
    setApplying(requested);
    setFailed(null);

    const apply = async () => {
      const res = await fetch("/api/dev/seed-scenario", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ scenario: requested }),
      });
      if (!res.ok) {
        throw new Error(`request rejected (${res.status})`);
      }

      // The watcher owns execution, so wait for the marker to catch up.
      for (let attempt = 0; attempt < 60; attempt++) {
        await new Promise((resolve) => setTimeout(resolve, 1000));
        if (cancelled) {
          return;
        }
        if ((await readActive()) === requested) {
          setActive(requested);
          setApplying(null);
          // A full reload, not router.refresh(): the script writes MySQL behind
          // the app's back, so every tRPC query is holding pre-seed data and
          // refreshing server components alone leaves things like the paused
          // banner stale. The param stays in the URL, and `handled` does not
          // survive the reload, so guard on the marker matching instead.
          window.location.reload();
          return;
        }
      }
      throw new Error("timed out waiting for seed-scenario.sh --watch");
    };

    apply().catch((err: Error) => {
      if (!cancelled) {
        setApplying(null);
        setFailed(err.message);
      }
    });

    return () => {
      cancelled = true;
    };
  }, [loaded, requested, active]);

  // Fill the URL in when a scenario was applied from a terminal, so the address
  // bar says which one is loaded. A param already present is authoritative and is
  // never overwritten — doing so races the apply above and cancels the switch.
  useEffect(() => {
    if (!active || applying || requested) {
      return;
    }
    handled.current.add(active);
    const url = new URL(window.location.href);
    url.searchParams.set("scenario", active);
    router.replace(`${url.pathname}${url.search}` as Route);
  }, [active, applying, requested, router]);

  const label = applying
    ? `seeding ${applying}…`
    : failed
      ? `seed failed: ${failed}`
      : active
        ? `seeded scenario: ${active}`
        : null;

  if (!label) {
    return null;
  }

  const colour = applying ? PENDING : failed ? "#b42318" : (COLOURS[active ?? ""] ?? FALLBACK);

  return (
    <div
      className="fixed inset-x-0 top-0 z-50 flex justify-center border-t-2"
      style={{ borderColor: colour }}
    >
      <div
        className="-mt-1 flex select-none items-center gap-2 overflow-hidden rounded-b px-1.5 py-0.5 font-mono text-white text-xs shadow-lg"
        style={{ backgroundColor: colour }}
      >
        {label}
      </div>
    </div>
  );
}
