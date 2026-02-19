"use client";
import { AnimatePresence, motion } from "framer-motion";
import { useEffect } from "react";
import { ProjectContentWrapper } from "../../../components/project-content-wrapper";
import { useProjectData } from "../../data-provider";
import { DeploymentDomainsSection } from "./(overview)/components/sections/deployment-domains-section";
import { DeploymentInfoSection } from "./(overview)/components/sections/deployment-info-section";
import { DeploymentNetworkSection } from "./(overview)/components/sections/deployment-network-section";
import { DeploymentProgressSection } from "./(overview)/components/sections/deployment-progress-section";
import { useDeployment } from "./layout-provider";

const fadeTransition = { duration: 0.3, ease: "easeOut" } as const;

export default function DeploymentOverview() {
  const { deploymentId } = useDeployment();
  const { getDeploymentById, refetchDomains } = useProjectData();
  const deployment = getDeploymentById(deploymentId);

  const ready = deployment?.status === "ready";

  useEffect(() => {
    if (ready) {
      refetchDomains();
    }
  }, [ready, refetchDomains]);

  return (
    <ProjectContentWrapper centered>
      <AnimatePresence mode="wait">
        {ready ? (
          <motion.div
            className="flex flex-col gap-5"
            key="ready"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={fadeTransition}
          >
            <DeploymentInfoSection />
            <DeploymentDomainsSection />
            <DeploymentNetworkSection />
          </motion.div>
        ) : (
          <motion.div
            className="flex flex-col gap-5"
            key="progress"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={fadeTransition}
          >
            <DeploymentProgressSection />
          </motion.div>
        )}
      </AnimatePresence>
    </ProjectContentWrapper>
  );
}
