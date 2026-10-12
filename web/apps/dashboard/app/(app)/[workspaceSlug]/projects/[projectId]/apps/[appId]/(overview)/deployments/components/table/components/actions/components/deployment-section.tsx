import type { DeploymentSummary } from "@/lib/collections";
import { DeploymentCard } from "./deployment-card";

type DeploymentSectionProps = {
  title: string;
  deployment: DeploymentSummary;
  isCurrent: boolean;
};

export const DeploymentSection = ({ title, deployment, isCurrent }: DeploymentSectionProps) => (
  <div className="flex flex-col gap-2">
    <h3 className="text-sm text-grayA-11">{title}</h3>
    <DeploymentCard deployment={deployment} isCurrent={isCurrent} />
  </div>
);
