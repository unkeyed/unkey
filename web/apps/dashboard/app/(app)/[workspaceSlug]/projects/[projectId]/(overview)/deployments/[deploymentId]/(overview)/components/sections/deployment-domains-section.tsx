"use client";

import { Earth } from "@unkey/icons";
import { useParams } from "next/navigation";
import { Section, SectionHeader } from "../../../../../../components/section";
import { DomainRow, DomainRowEmpty, DomainRowSkeleton } from "../../../../../details/domain-row";
import { useProjectData } from "../../../../../data-provider";

export function DeploymentDomainsSection() {
  const params = useParams();
  const deploymentId = params?.deploymentId as string;

  const { getDomainsForDeployment, isDomainsLoading } = useProjectData();
  const domains = getDomainsForDeployment(deploymentId);
  return (
    <Section>
      <SectionHeader
        icon={<Earth iconSize="md-regular" className="text-gray-9" />}
        title="Domains"
      />
      <div>
        {isDomainsLoading ? (
          <>
            <DomainRowSkeleton />
            <DomainRowSkeleton />
          </>
        ) : domains.length > 0 ? (
          domains.map((domain) => (
            <DomainRow key={domain.id} domain={domain.fullyQualifiedDomainName} />
          ))
        ) : (
          <DomainRowEmpty
            title="No domains found"
            description="Your configured domains will appear here once they're set up and verified."
          />
        )}
      </div>
    </Section>
  );
}
