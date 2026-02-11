import { InfoTooltip } from "@unkey/ui";
import { useProjectData } from "../../../../data-provider";
import type { DeploymentStatus } from "../../../filters.schema";
import { DomainListSkeleton } from "./skeletons";

type Props = {
  deploymentId: string;
  status: DeploymentStatus;
};

export const DomainList = ({ deploymentId, status }: Props) => {
  const { getDomainsForDeployment, isDomainsLoading } = useProjectData();

  // Show placeholder for failed deployments
  if (status === "failed") {
    return <span className="text-xs text-gray-9 font-mono">—</span>;
  }

  // Only show skeleton when actually loading
  if (isDomainsLoading) {
    return <DomainListSkeleton />;
  }

  // Get domains for this deployment and sort client-side
  const domainsForDeployment = getDomainsForDeployment(deploymentId);
  const sortedDomains = [...domainsForDeployment].sort((a, b) =>
    a.fullyQualifiedDomainName.localeCompare(b.fullyQualifiedDomainName),
  );

  // Handle empty domains (valid for non-failed deployments)
  if (!sortedDomains.length) {
    return <span className="text-xs text-gray-9 font-mono">—</span>;
  }

  // Always show environment domain first, fallback to first domain if none
  const environmentDomain = sortedDomains.find((d) => d.sticky === "environment");
  const primaryDomain = environmentDomain ?? sortedDomains[0];
  const additionalDomains = sortedDomains.filter((d) => d.id !== primaryDomain.id);

  // Single domain case - no tooltip needed
  if (sortedDomains.length === 1) {
    return (
      <a
        href={`https://${primaryDomain.fullyQualifiedDomainName}`}
        target="_blank"
        rel="noopener noreferrer"
        className="text-accent-12 text-xs font-mono hover:underline decoration-dashed underline-offset-2 transition-all truncate block max-w-[200px]"
        onClick={(e) => e.stopPropagation()}
      >
        {primaryDomain.fullyQualifiedDomainName}
      </a>
    );
  }

  // Multiple domains: show primary + count badge with tooltip
  return (
    <div className="flex items-center gap-2 min-w-0">
      <a
        href={`https://${primaryDomain.fullyQualifiedDomainName}`}
        target="_blank"
        rel="noopener noreferrer"
        className="text-accent-12 text-xs font-mono hover:underline decoration-dashed underline-offset-2 transition-all truncate block max-w-[150px]"
        onClick={(e) => e.stopPropagation()}
      >
        {primaryDomain.fullyQualifiedDomainName}
      </a>

      <InfoTooltip
        position={{
          side: "bottom",
          align: "start",
        }}
        content={
          <div className="space-y-2 max-w-[300px] py-2">
            {additionalDomains.map((d) => (
              <div key={d.id} className="text-xs font-medium flex items-center gap-1.5">
                <div className="w-1 h-1 bg-gray-8 rounded-full shrink-0" />
                <a
                  href={`https://${d.fullyQualifiedDomainName}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="transition-all hover:underline decoration-dashed underline-offset-2"
                  onClick={(e) => e.stopPropagation()}
                >
                  {d.fullyQualifiedDomainName}
                </a>
              </div>
            ))}
          </div>
        }
      >
        <div className="rounded-full px-1.5 py-0.5 bg-grayA-3 text-gray-12 text-xs leading-[18px] font-mono tabular-nums cursor-pointer hover:bg-grayA-4 transition-colors">
          +{additionalDomains.length}
        </div>
      </InfoTooltip>
    </div>
  );
};
