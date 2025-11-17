import { DomainRow } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/details/domain-row";
import { CircleInfo } from "@unkey/icons";

type DomainsSectionProps = {
  domains: Array<{ id: string; hostname: string }>;
};

export const DomainsSection = ({ domains }: DomainsSectionProps) => {
  if (domains.length === 0) {
    return null;
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <h3 className="text-[13px] text-grayA-11">
          {domains.length === 1 ? "Domain" : "Domains"}
        </h3>
        <CircleInfo iconSize="sm-regular" className="text-gray-9" />
      </div>
      <div>
        {domains.map((domain) => (
          <DomainRow
            key={domain.id}
            domain={domain.hostname}
            className="bg-white dark:bg-black border-grayA-5 first:rounded-t-lg last:rounded-b-lg"
          />
        ))}
      </div>
    </div>
  );
};
