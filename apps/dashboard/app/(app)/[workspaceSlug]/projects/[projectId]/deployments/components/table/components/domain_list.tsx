import { eq, useLiveQuery } from "@tanstack/react-db";
import { useProject } from "../../../../layout-provider";

type Props = {
  deploymentId: string;
};

export const DomainList = ({ deploymentId }: Props) => {
  const { collections } = useProject();
  const domains = useLiveQuery((q) =>
    q
      .from({ domain: collections.domains })
      .where(({ domain }) => eq(domain.deploymentId, deploymentId))
      .orderBy(({ domain }) => domain.domain, "asc"),
  );

  if (domains.isLoading || !domains.data.length) {
    return <DomainListSkeleton />;
  }

  return (
    <ul className="flex flex-col list-none py-2">
      {domains.data.map((domain) => (
        <li key={domain.id}>https://{domain.domain}</li>
      ))}
    </ul>
  );
};

const DomainListSkeleton = () => (
  <ul className="flex flex-col list-none py-2 gap-1">
    {[1, 2, 3].map((i) => (
      <li key={i}>
        <div className="h-3 w-64 bg-grayA-3 rounded animate-pulse" />
      </li>
    ))}
  </ul>
);
