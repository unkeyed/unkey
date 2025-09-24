import { eq, useLiveQuery } from "@tanstack/react-db";
import { useProjectLayout } from "../../../../layout-provider";

type Props = {
  deploymentId: string;
  // I couldn't figure out how to make the domains revalidate on a rollback
  // From my understanding it should already work, because we're using the
  // .util.refetch() in the trpc mutation, but it doesn't.
  // We need to investigate this later
  hackyRevalidateDependency?: unknown;
};

export const DomainList = ({ deploymentId, hackyRevalidateDependency }: Props) => {
  const { collections } = useProjectLayout();
  const domains = useLiveQuery(
    (q) =>
      q
        .from({ domain: collections.domains })
        .where(({ domain }) => eq(domain.deploymentId, deploymentId))
        .orderBy(({ domain }) => domain.domain, "asc"),
    [hackyRevalidateDependency],
  );

  return (
    <ul className="flex flex-col list-none py-2">
      {domains.data.map((domain) => (
        <li key={domain.id}>https://{domain.domain}</li>
      ))}
    </ul>
  );
};
