"use client";

import type { Deployment } from "@/lib/collections";
import { routes } from "@/lib/navigation/routes";
import { Card } from "@unkey/ui";
import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import { useAppScope } from "../../environment-context";
import { AppCanvas, AppNode } from "./app-canvas";
import { NewerDeploymentRow } from "./card-newer-deployment";
import { ProductionCardRollbackBanner } from "./card-rollback-banner";
import { type CardDomain, useProductionCard } from "./production-card-context";

export function AppProductionCard({
  domains,
  newerDeployment,
}: {
  domains: CardDomain[];
  newerDeployment: Deployment | undefined;
}) {
  const scope = useAppScope();
  const { isRolledBack } = useProductionCard();
  const reduceMotion = useReducedMotion();

  return (
    <div className="relative">
      {isRolledBack && <ProductionCardRollbackBanner />}
      <Card className="relative z-10 flex flex-col">
        <AppCanvas domains={domains} app={<AppNode />} />
        <AnimatePresence initial={false} mode="wait">
          {newerDeployment && (
            <motion.div
              key={newerDeployment.id}
              className="overflow-hidden"
              initial={{ height: 0, opacity: 0 }}
              animate={{ height: "auto", opacity: 1 }}
              exit={{ height: 0, opacity: 0 }}
              transition={
                reduceMotion ? { duration: 0 } : { duration: 0.2, ease: [0.215, 0.61, 0.355, 1] }
              }
            >
              <NewerDeploymentRow
                deployment={newerDeployment}
                href={routes.projects.apps.deployment({
                  ...scope,
                  deploymentId: newerDeployment.id,
                  build: true,
                })}
              />
            </motion.div>
          )}
        </AnimatePresence>
      </Card>
    </div>
  );
}
