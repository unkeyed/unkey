import { eq, useLiveQuery } from "@tanstack/react-db";
import { InfoTooltip } from "@unkey/ui";
import { useProject } from "../../../../layout-provider";
import { DomainListSkeleton } from "./skeletons";

type Props = {
  deploymentId: string;
};

export const DomainList = ({ deploymentId }: Props) => {
  const { collections } = useProject();

  const domains = useLiveQuery((q) =>
    q
      .from({ domain: collections.domains })
      .where(({ domain }) => eq(domain.deploymentId, deploymentId))
      .orderBy(({ domain }) => domain.fullyQualifiedDomainName, "asc"),
  );

  if (domains.isLoading || !domains.data.length) {
    return <DomainListSkeleton />;
  }

  // Always show environment domain first, fallback to first domain if none
  const environmentDomain = domains.data.find((d) => d.sticky === "environment");
  const primaryDomain = environmentDomain ?? domains.data[0];
  const additionalDomains = domains.data.filter((d) => d.id !== primaryDomain.id);

  // Single domain case - no tooltip needed
  if (domains.data.length === 1) {
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
