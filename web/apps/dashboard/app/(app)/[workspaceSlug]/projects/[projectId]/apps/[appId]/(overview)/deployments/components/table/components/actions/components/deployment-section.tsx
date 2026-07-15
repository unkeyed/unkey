import type { Deployment } from "@/lib/collections";
import { DeploymentCard } from "./deployment-card";

type DeploymentSectionProps = {
  title: string;
  deployment: Deployment;
  isCurrent: boolean;
};

export const DeploymentSection = ({ title, deployment, isCurrent }: DeploymentSectionProps) => (
  <div className="flex flex-col gap-2">
    <h3 className="text-[13px] text-grayA-11">{title}</h3>
    <DeploymentCard deployment={deployment} isCurrent={isCurrent} />
  </div>
);
