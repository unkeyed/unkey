"use client";

import { getCurrentVariant } from "@/hooks/use-navbar-variant";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { useLiveQuery } from "@tanstack/react-db";
import { Cube, Nodes } from "@unkey/icons";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";

type HomeCard = {
  id: string;
  name: string;
  kind: "project" | "api";
  href: string;
  secondary: string;
};

export default function HomePage() {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  // Read the variant from localStorage post-mount. Doing this inside useEffect
  // avoids the SSR flicker where the React state seed returns "current" and
  // then redirects before hydration has a chance to sync the real value.
  const [allowed, setAllowed] = useState(false);

  useEffect(() => {
    // Home exists only in the prototype variants (v1a + v1b). The production
    // `current` variant has no Home entry, so we redirect those users to the
    // workspace default rather than render an orphaned page.
    if (getCurrentVariant() === "current") {
      router.replace(`/${workspace.slug}`);
      return;
    }
    setAllowed(true);
  }, [router, workspace.slug]);

  const projectsQuery = useLiveQuery((q) =>
    q.from({ project: collection.projects }).select(({ project }) => ({
      id: project.id,
      name: project.name,
    })),
  );

  const apisQuery = trpc.api.overview.query.useInfiniteQuery(
    { limit: 18 },
    { getNextPageParam: (last) => last.nextCursor },
  );

  const cards = useMemo<HomeCard[]>(() => {
    const base = `/${workspace.slug}`;
    const projects: HomeCard[] = (projectsQuery.data ?? []).map((p) => ({
      id: p.id,
      name: p.name,
      kind: "project",
      href: `${base}/projects/${p.id}`,
      secondary: p.id,
    }));
    const apis: HomeCard[] = (apisQuery.data?.pages[0]?.apiList ?? []).map((a) => ({
      id: a.id,
      name: a.name,
      kind: "api",
      href: `${base}/apis/${a.id}`,
      secondary: a.id,
    }));
    return [...projects, ...apis];
  }, [projectsQuery.data, apisQuery.data, workspace.slug]);

  if (!allowed) {
    return null;
  }

  return (
    <div className="mx-auto w-full max-w-[1100px] px-8 pb-16 pt-10">
      <div className="mb-8 flex items-end justify-between">
        <div>
          <h1 className="text-[22px] font-semibold text-gray-12">Workspace</h1>
          <p className="text-[13px] text-gray-11">
            Everything in this workspace, deployed and authenticated.
          </p>
        </div>
      </div>

      {cards.length === 0 ? (
        <div className="rounded-xl border border-gray-4 p-8 text-center text-[13px] text-gray-11">
          Nothing here yet. Create a project or an API to get started.
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {cards.map((card) => (
            <Link
              key={`${card.kind}:${card.id}`}
              href={card.href}
              className="flex flex-col gap-3 rounded-xl border border-gray-4 bg-background p-4 transition-colors hover:border-gray-7"
            >
              <div className="flex items-center gap-3">
                <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-gray-3 text-gray-12">
                  {card.kind === "project" ? (
                    <Cube iconSize="sm-medium" />
                  ) : (
                    <Nodes iconSize="sm-medium" />
                  )}
                </div>
                <div className="flex min-w-0 flex-col">
                  <span className="truncate text-[13px] font-semibold text-gray-12">
                    {card.name}
                  </span>
                  <span className="truncate font-mono text-[11px] text-gray-11">
                    {card.secondary}
                  </span>
                </div>
              </div>
              <div className="mt-auto text-[11px] uppercase tracking-wide text-gray-9">
                {card.kind === "project" ? "Project" : "API"}
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
