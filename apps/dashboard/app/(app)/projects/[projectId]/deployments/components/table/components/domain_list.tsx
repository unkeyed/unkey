import { and, eq, or, useLiveQuery } from "@tanstack/react-db";
import { useProjectLayout } from "../../../../layout-provider";

type Props = {
  deploymentId: string;
};

export const DomainList = ({ deploymentId }: Props) => {
  const { collections } = useProjectLayout();

  const domains = useLiveQuery((q) =>
    q
      .from({ domain: collections.domains })
      .where(({ domain }) =>
        or(
          eq(domain.rolledBackDeploymentId, deploymentId),
          and(eq(domain.deploymentId, deploymentId), eq(domain.rolledBackDeploymentId, null)),
        ),
      ),
  );

  return (
    <ul className="flex flex-col list-none">
      {domains.data
        .sort((a, b) => b.domain.localeCompare(a.domain))
        .map((domain) => (
          <li key={domain.id}>https://{domain.domain}</li>
        ))}
    </ul>
  );
};
