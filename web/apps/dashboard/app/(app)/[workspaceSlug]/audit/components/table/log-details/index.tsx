"use client";

import { LogDetails } from "@/components/logs/details/log-details";
import type { AuditLog } from "@/lib/trpc/routers/audit/schema";
import { LogFooter } from "./components/log-footer";
import { LogHeader } from "./components/log-header";
import { LogSection } from "./components/log-section";

const ANIMATION_DELAY = 350;

type Props = {
  distanceToTop: number;
  selectedLog: AuditLog | null;
  setSelectedLog: (log: AuditLog | null) => void;
};

export const AuditLogDetails = ({ distanceToTop, selectedLog, setSelectedLog }: Props) => {
  const handleClose = () => {
    setSelectedLog(null);
  };

  if (!selectedLog) {
    return null;
  }

  const sections = [
    <LogFooter key="footer" log={selectedLog} />,
    ...selectedLog.auditLog.targets.map((target) => {
      const title = String(target.type).charAt(0).toUpperCase() + String(target.type).slice(1);

      return <LogSection key={target.id} details={JSON.stringify(target, null, 2)} title={title} />;
    }),
  ].filter(Boolean);

  return (
    <LogDetails distanceToTop={distanceToTop} log={selectedLog} onClose={handleClose}>
      <LogDetails.Header onClose={handleClose}>
        <LogHeader log={selectedLog} onClose={handleClose} />
      </LogDetails.Header>
      <LogDetails.CustomSections startDelay={150} staggerDelay={50}>
        {sections}
      </LogDetails.CustomSections>
      <LogDetails.Spacer delay={ANIMATION_DELAY} />
    </LogDetails>
  );
};
