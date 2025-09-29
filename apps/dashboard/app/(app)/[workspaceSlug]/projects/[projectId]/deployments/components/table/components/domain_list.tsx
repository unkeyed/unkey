import { eq, useLiveQuery } from "@tanstack/react-db";
import { useProjectLayout } from "../../../../layout-provider";

type Props = {
  deploymentId: string;
};

export const DomainList = ({ deploymentId }: Props) => {
  const { collections } = useProjectLayout();
  const domains = useLiveQuery((q) =>
    q
      .from({ domain: collections.domains })
      .where(({ domain }) => eq(domain.deploymentId, deploymentId))
      .orderBy(({ domain }) => domain.domain, "asc"),
  );

  return (
    <ul className="flex flex-col list-none py-2">
      {domains.data.map((domain) => (
        <li key={domain.id}>https://{domain.domain}</li>
      ))}
    </ul>
  );
};
