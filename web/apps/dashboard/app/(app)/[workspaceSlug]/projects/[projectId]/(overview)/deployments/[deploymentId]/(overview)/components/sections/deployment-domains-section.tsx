"use client";

import { eq, useLiveQuery } from "@tanstack/react-db";
import { Earth } from "@unkey/icons";
import { useParams } from "next/navigation";
import { Section, SectionHeader } from "../../../../../../components/section";
import {
  DomainRow,
  DomainRowEmpty,
  DomainRowSkeleton,
} from "../../../../../details/domain-row";
import { useProject } from "../../../../../layout-provider";

export function DeploymentDomainsSection() {
  const params = useParams();
  const deploymentId = params?.deploymentId as string;

  const { collections } = useProject();

  const { data: domains, isLoading } = useLiveQuery(
    (q) =>
      q
        .from({ domain: collections.domains })
        .where(({ domain }) => eq(domain.deploymentId, deploymentId)),
    [deploymentId],
  );

  return (
    <Section>
      <SectionHeader
        icon={<Earth iconSize="md-regular" className="text-gray-9" />}
        title="Domains"
      />
      <div>
        {isLoading ? (
          <>
            <DomainRowSkeleton />
            <DomainRowSkeleton />
          </>
        ) : (domains?.length ?? 0) > 0 ? (
          domains?.map((domain) => (
            <DomainRow
              key={domain.id}
              domain={domain.fullyQualifiedDomainName}
            />
          ))
        ) : (
          <DomainRowEmpty />
        )}
      </div>
    </Section>
  );
}
