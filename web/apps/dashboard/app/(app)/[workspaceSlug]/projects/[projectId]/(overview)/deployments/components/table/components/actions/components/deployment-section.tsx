import type { Deployment } from "@/lib/collections";
import { CircleInfo } from "@unkey/icons";
import { DeploymentCard } from "./deployment-card";

type DeploymentSectionProps = {
  title: string;
  deployment: Deployment;
  isCurrent: boolean;
  showSignal?: boolean;
};

export const DeploymentSection = ({
  title,
  deployment,
  isCurrent,
  showSignal,
}: DeploymentSectionProps) => (
  <div className="flex flex-col gap-2">
    <div className="flex items-center gap-2">
      <h3 className="text-[13px] text-grayA-11">{title}</h3>
      <CircleInfo iconSize="sm-regular" className="text-gray-9" />
    </div>
    <DeploymentCard deployment={deployment} isCurrent={isCurrent} showSignal={showSignal} />
  </div>
);
