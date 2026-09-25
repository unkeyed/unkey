"use client";

import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { useAppHomeHref } from "@/hooks/use-app-home-href";
import { useProject } from "@/hooks/use-project";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { projectDisplayName } from "@/lib/collections/deploy/projects";
import { routes } from "@/lib/navigation/routes";
import { buildRoute } from "@/lib/navigation/routes/shared";
import { trpc } from "@/lib/trpc/client";
import type { ProjectOverview } from "@/lib/trpc/routers/deploy/project/overview";
import { cn } from "@unkey/ui/src/lib/utils";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { type ComponentType, useEffect, useLayoutEffect, useRef, useState } from "react";
import { buildOverviewModel } from "../../overview/_components/overview-model";
import { SCENARIOS } from "../../overview/_components/scenario-switcher";
import type { CanvasActions, CanvasLinks } from "./proto-canvas";
import type { VariantProps } from "./shared";
import { useTotals } from "./totals";
import { AsideVariant } from "./variants/aside";
import { BleedVariant } from "./variants/bleed";
import { CommandVariant } from "./variants/command";
import { FlowVariant } from "./variants/flow";
import { InspectorVariant } from "./variants/inspector";
import { LanesVariant } from "./variants/lanes";
import { ReadoutVariant } from "./variants/readout";

const VARIANTS: Array<{ name: string; Component: ComponentType<VariantProps> }> = [
  { name: "Aside", Component: AsideVariant },
  { name: "Bleed", Component: BleedVariant },
  { name: "Flow", Component: FlowVariant },
  { name: "Readout", Component: ReadoutVariant },
  { name: "Inspector", Component: InspectorVariant },
  { name: "Lanes", Component: LanesVariant },
  { name: "Command", Component: CommandVariant },
];

export function Harness() {
  const params = useParams();
  const projectId = typeof params?.projectId === "string" ? params.projectId : "";
  const { data, isLoading, error } = trpc.deploy.project.overview.useQuery(
    { projectId },
    { enabled: Boolean(projectId), refetchInterval: 10_000 },
  );
  const search = useSearchParams();
  const initial = Math.min(
    Math.max((Number.parseInt(search?.get("v") ?? "1", 10) || 1) - 1, 0),
    VARIANTS.length - 1,
  );
  const [current, setCurrent] = useState(initial);

  return (
    <div className="h-full">
      {data ? (
        <Loaded data={data} current={current} />
      ) : (
        <div className="p-8 text-sm text-gray-9">
          {isLoading ? "Loading project…" : (error?.message ?? "No data")}
        </div>
      )}
      <Picker current={current} onChange={setCurrent} />
      <ProtoScenarios current={projectId} variant={current} />
    </div>
  );
}

function Loaded({ data, current }: { data: ProjectOverview; current: number }) {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const { project } = useProject();
  const appHomeHref = useAppHomeHref();
  const { gated, openPaywall, planGate } = useDeployActionGate();
  const model = buildOverviewModel(data);
  const totals = useTotals(data);
  const scope = { workspaceSlug: workspace.slug, projectId: data.project.id };

  const links: CanvasLinks = {
    app: (appId) => appHomeHref({ ...scope, appId }),
    allApps: routes.projects.detail(scope),
    keyspace: (apiId) => routes.apis.detail({ ...scope, apiId }),
    allKeyspaces: routes.apis.list(scope),
    ratelimit: (namespaceId) => routes.ratelimits.detail({ ...scope, namespaceId }),
    allRatelimits: routes.ratelimits.list(scope),
  };
  const actions: CanvasActions = {
    createApp: () => (gated ? openPaywall() : router.push(routes.projects.apps.new(scope))),
    createKeyspace: () => router.push(routes.apis.list({ ...scope, new: true })),
    createRatelimit: () => router.push(routes.ratelimits.list(scope)),
    openIdentities: () => router.push(routes.identities.list(scope)),
    openPermissions: () => router.push(routes.authorization.roles(scope)),
  };
  const title = project ? projectDisplayName(project, workspace.name) : data.project.name;
  const { Component } = VARIANTS[current];

  return (
    <>
      <Component
        key={current}
        data={data}
        model={model}
        links={links}
        actions={actions}
        title={title}
        crumb={`${workspace.slug} / ${data.project.slug}`}
        totals={totals}
      />
      {planGate}
    </>
  );
}

function protoRoute(workspaceSlug: string, projectId: string, v: number) {
  return buildRoute(
    "/[workspaceSlug]/projects/[projectId]/overview-prototypes",
    { workspaceSlug, projectId },
    { v: String(v + 1) },
  );
}

function ProtoScenarios({ current, variant }: { current: string; variant: number }) {
  const [open, setOpen] = useState(false);
  const groups = [...new Set(SCENARIOS.map((s) => s.group))];
  return (
    <div className="fixed right-4 bottom-4 z-40 flex flex-col items-end gap-2">
      {open && (
        <div className="w-80 overflow-hidden rounded-xl border border-border bg-raised p-1.5 shadow-floating">
          {groups.map((g) => (
            <div key={g} className="mb-1">
              <div className="px-2 pt-2 pb-1 text-[10px] font-medium uppercase tracking-wide text-gray-9">
                {g}
              </div>
              {SCENARIOS.filter((s) => s.group === g).map((s) => (
                <a
                  key={s.projectId}
                  href={routes.auth.switchOrganization({
                    organizationId: s.orgId,
                    returnTo: protoRoute(s.workspaceSlug, s.projectId, variant),
                  })}
                  className={cn(
                    "flex items-center justify-between gap-2 rounded-md px-2 py-1.5 text-[13px] text-gray-12 hover:bg-grayA-3",
                    s.projectId === current && "bg-grayA-3",
                  )}
                >
                  <span>{s.label}</span>
                  <span className="text-[11px] text-gray-9">{s.note}</span>
                </a>
              ))}
            </div>
          ))}
        </div>
      )}
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="rounded-full border border-border bg-raised px-3 py-1.5 text-[11px] font-medium text-gray-11 shadow-floating hover:text-gray-12"
      >
        Scenarios
      </button>
    </div>
  );
}

const PICKER_CSS = `
.proto-picker {
  position: fixed;
  bottom: 24px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 2147483647;
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 4px;
  border-radius: 999px;
  background: rgba(10, 10, 10, 0.82);
  -webkit-backdrop-filter: blur(12px) saturate(1.4);
  backdrop-filter: blur(12px) saturate(1.4);
  box-shadow:
    0 0 0 1px rgba(255, 255, 255, 0.08) inset,
    0 8px 24px rgba(0, 0, 0, 0.24),
    0 2px 6px rgba(0, 0, 0, 0.12);
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  font-size: 13px;
  line-height: 1;
  -webkit-font-smoothing: antialiased;
  user-select: none;
  -webkit-user-select: none;
}
.proto-picker-highlight {
  position: absolute;
  top: 4px;
  left: 0;
  height: 28px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.12);
  will-change: transform;
}
.proto-picker[data-ready] .proto-picker-highlight {
  transition:
    transform 250ms cubic-bezier(0.23, 1, 0.32, 1),
    width 250ms cubic-bezier(0.23, 1, 0.32, 1);
}
@media (prefers-reduced-motion: reduce) {
  .proto-picker[data-ready] .proto-picker-highlight { transition: none; }
}
.proto-picker-item {
  position: relative;
  display: flex;
  align-items: center;
  height: 28px;
  padding: 0 12px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: rgba(255, 255, 255, 0.55);
  font: inherit;
  cursor: pointer;
  transition: color 150ms ease-out;
}
.proto-picker-item:hover { color: rgba(255, 255, 255, 0.85); }
.proto-picker-item:active { transform: scale(0.97); }
.proto-picker-item:focus-visible { outline: 2px solid rgba(255, 255, 255, 0.4); outline-offset: 2px; }
.proto-picker-item[data-active] { color: #fff; }
.proto-picker-divider { width: 1px; height: 16px; margin: 0 4px; background: rgba(255, 255, 255, 0.12); }
.proto-picker-replay { padding: 0 10px; font-size: 14px; }
.proto-picker[data-position="top"] { bottom: auto; top: 24px; }
`;

function Picker({ current, onChange }: { current: number; onChange: (i: number) => void }) {
  const navRef = useRef<HTMLElement>(null);
  const highlightRef = useRef<HTMLSpanElement>(null);
  const itemRefs = useRef<Array<HTMLButtonElement | null>>([]);
  const [ready, setReady] = useState(false);

  useLayoutEffect(() => {
    const move = () => {
      const el = itemRefs.current[current];
      const hl = highlightRef.current;
      if (!el || !hl) {
        return;
      }
      hl.style.width = `${el.offsetWidth}px`;
      hl.style.transform = `translateX(${el.offsetLeft}px)`;
    };
    move();
    const url = new URL(window.location.href);
    url.searchParams.set("v", String(current + 1));
    window.history.replaceState(window.history.state, "", url);
    window.addEventListener("resize", move);
    return () => window.removeEventListener("resize", move);
  }, [current]);

  useEffect(() => {
    const id = requestAnimationFrame(() => requestAnimationFrame(() => setReady(true)));
    return () => cancelAnimationFrame(id);
  }, []);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      const t = e.target;
      if (
        t instanceof HTMLElement &&
        (/^(INPUT|TEXTAREA|SELECT)$/.test(t.tagName) || t.isContentEditable)
      ) {
        return;
      }
      if (e.metaKey || e.ctrlKey || e.altKey) {
        return;
      }
      const n = Number.parseInt(e.key, 10);
      if (n >= 1 && n <= VARIANTS.length) {
        onChange(n - 1);
      } else if (e.key === "ArrowRight") {
        onChange((current + 1) % VARIANTS.length);
      } else if (e.key === "ArrowLeft") {
        onChange((current - 1 + VARIANTS.length) % VARIANTS.length);
      }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [current, onChange]);

  return (
    <>
      <style>{PICKER_CSS}</style>
      <nav
        ref={navRef}
        className="proto-picker"
        aria-label="Prototype variants"
        data-position="top"
        data-ready={ready ? "" : undefined}
      >
        <span ref={highlightRef} className="proto-picker-highlight" aria-hidden="true" />
        {VARIANTS.map((v, i) => (
          <button
            key={v.name}
            type="button"
            ref={(el) => {
              itemRefs.current[i] = el;
            }}
            className="proto-picker-item"
            data-active={i === current ? "" : undefined}
            aria-current={i === current ? "true" : undefined}
            onClick={() => onChange(i)}
          >
            {v.name}
          </button>
        ))}
      </nav>
    </>
  );
}
