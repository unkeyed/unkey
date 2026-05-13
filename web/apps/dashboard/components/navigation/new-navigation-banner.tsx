"use client";

import { useDismissibleBanner } from "@/hooks/use-dismissible-banner";
import { useFlag } from "@/lib/flags/provider";
import { Badge, BannerCard, Button } from "@unkey/ui";
import { motion } from "framer-motion";
import Link from "next/link";

const CHANGELOG_URL = "https://unkey.com/changelog";

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
  const enabled = useFlag("newNavigation");
  const { dismissed, dismiss } = useDismissibleBanner("new-navigation-v1");

  if (!enabled || dismissed) {
    return null;
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 16 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ type: "spring", stiffness: 300, damping: 25 }}
      className="fixed bottom-4 right-4 z-40 w-85"
    >
      <BannerCard onDismiss={dismiss} illustration={<NewNavigationIllustration />}>
        <div className="flex flex-col gap-3">
          <Badge variant="success" className="self-start">
            Beta
          </Badge>
          <div className="flex flex-col gap-1 pr-6">
            <p className="text-sm font-medium text-content">Dashboard, upgraded.</p>
            <p className="text-xs text-content-subtle">
              You're trying a new way of navigating Unkey. It's still in beta — let us know what
              you think.
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
