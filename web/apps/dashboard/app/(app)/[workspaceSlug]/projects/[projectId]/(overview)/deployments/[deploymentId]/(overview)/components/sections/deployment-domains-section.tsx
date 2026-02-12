"use client";

import { Earth } from "@unkey/icons";
import { Section, SectionHeader } from "../../../../../../components/section";
import { EmptySection } from "../../../../../components/empty-section";
import { useProjectData } from "../../../../../data-provider";
import { DomainRow, DomainRowSkeleton } from "../../../../../details/domain-row";
import { useDeployment } from "../../../layout-provider";

export function DeploymentDomainsSection() {
  const { deploymentId } = useDeployment();
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
          <EmptySection
            title="No domains found"
            description="Your configured domains will appear here once they're set up and verified."
          />
        )}
      </div>
    </Section>
  );
}
