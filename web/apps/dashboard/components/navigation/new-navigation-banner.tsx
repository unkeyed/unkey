"use client";

import { useDismissibleBanner } from "@/hooks/use-dismissible-banner";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { Badge, BannerCard, Button } from "@unkey/ui";
import { motion, useReducedMotion } from "framer-motion";
import Link from "next/link";

const CHANGELOG_URL = "https://www.unkey.com/changelog#2026-06-01";

function NewNavigationIllustration() {
  return (
    <>
      <img
        src="/images/ascii-noise-1.png"
        alt=""
        aria-hidden
        className="absolute -right-12 -top-8 max-w-none origin-top-right scale-60 opacity-30 grayscale dark:invert"
      />
      <div className="absolute inset-0 bg-gradient-to-tr from-background via-background/80 to-transparent" />
    </>
  );
}

export function NewNavigationBanner() {
  const workspace = useWorkspaceNavigation();
  const { dismissed, dismiss } = useDismissibleBanner("new-navigation-v1");
  const reduceMotion = useReducedMotion();

  // The banner reassures users that their APIs moved under Keyspaces, so it
  // only makes sense for workspaces that actually have APIs. A workspace with
  // none has nothing that got renamed (e.g. a fresh signup on an empty
  // projects page), so we skip the query and the banner entirely.
  const { data } = trpc.api.overview.query.useQuery({ limit: 1 }, { enabled: !dismissed });
  const hasApis = (data?.total ?? 0) > 0;

  if (dismissed || !hasApis) {
    return null;
  }

  return (
    <motion.div
      initial={reduceMotion ? false : { opacity: 0, y: 16 }}
      animate={{ opacity: 1, y: 0 }}
      transition={reduceMotion ? { duration: 0 } : { type: "spring", stiffness: 300, damping: 25 }}
      className="fixed bottom-4 right-4 z-40 w-85"
    >
      <BannerCard onDismiss={dismiss} illustration={<NewNavigationIllustration />}>
        <div className="flex flex-col gap-3">
          <Badge variant="success" className="self-start">
            New
          </Badge>
          <div className="flex flex-col gap-1 pr-6">
            <p className="text-sm font-medium text-content">Meet the new Unkey</p>
            <p className="text-xs text-content-subtle">
              A cleaner dashboard, built around Projects. Your APIs now live in{" "}
              <Link
                href={routes.apis.list({ workspaceSlug: workspace.slug })}
                onClick={dismiss}
                className="text-content underline underline-offset-2 hover:text-content/80"
              >
                Keyspaces (APIs)
              </Link>
              , with the same keys and data.
            </p>
          </div>
          <Button variant="outline" size="sm" asChild className="self-start">
            <Link href={CHANGELOG_URL} target="_blank" rel="noreferrer">
              Learn more
            </Link>
          </Button>
        </div>
      </BannerCard>
    </motion.div>
  );
}
