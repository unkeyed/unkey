import { Cube } from "@unkey/icons";
import { Button, InfoTooltip } from "@unkey/ui";
import { useProjectData } from "../../data-provider";
import { DetailSection } from "./detail-section";
import { createDetailSections } from "./sections";

export const ProjectDetailsContent = () => {
  const { getDomainsForDeployment, project, getDeploymentById } = useProjectData();

  const deployment = project?.liveDeploymentId
    ? getDeploymentById(project.liveDeploymentId)
    : undefined;

  const data = project && deployment
    ? { project, deployment }
    : undefined;

  // Get domains from provider and transform
  const domainsData = data?.project.liveDeploymentId
    ? getDomainsForDeployment(data.project.liveDeploymentId)
        .map((d) => ({
          domain: d.fullyQualifiedDomainName,
          environment: d.sticky,
        }))
        .sort((a, b) => a.domain.localeCompare(b.domain))
    : [];

  if (!data?.deployment) {
    return null;
  }

  const detailSections = createDetailSections({
    ...data.deployment,
    repository: null,
  });

  // This "environment" domain never changes even when you do a rollback this one stays stable.
  const mainDomain = domainsData.find((d) => d.environment === "environment")?.domain;
  const gitShaAndBranchNameDomains = domainsData.filter((d) => d.environment !== "environment");

  return (
    <>
      {/* Domains Section */}
      <div className="h-20 mt-4 px-4">
        <div className="items-center flex gap-4">
          <Button
            variant="outline"
            className="size-12 p-0 bg-grayA-3 border border-grayA-3 rounded-xl"
          >
            <Cube iconSize="2xl-medium" className="!size-[20px]" />
          </Button>
          <div className="flex flex-col gap-1">
            <span className="text-accent-12 font-medium text-sm truncate">{data.project.name}</span>
            <div className="gap-2 items-center flex min-w-0 max-w-[250px]">
              {/* # is okay. This section is not accessible without deploy*/}
              <a
                href={mainDomain ?? "#"}
                target="_blank"
                rel="noopener noreferrer"
                className="text-gray-9 text-sm truncate block transition-all hover:underline decoration-dashed underline-offset-2"
              >
                {mainDomain}
              </a>
              <InfoTooltip
                position={{
                  side: "bottom",
                }}
                content={
                  <div className="space-y-2 max-w-[300px] py-2">
                    {gitShaAndBranchNameDomains.map((d) => (
                      <div key={d.domain} className="text-xs font-medium flex items-center gap-1.5">
                        <div className="w-1 h-1 bg-gray-8 rounded-full shrink-0" />
                        <a
                          href={`https://${d.domain}`}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="transition-all hover:underline decoration-dashed underline-offset-2"
                        >
                          {d.domain}
                        </a>
                      </div>
                    ))}
                  </div>
                }
              >
                <div className="rounded-full px-1.5 py-0.5 bg-grayA-3 text-gray-12 text-xs leading-[18px] font-mono tabular-nums">
                  +{gitShaAndBranchNameDomains.length}
                </div>
              </InfoTooltip>
            </div>
          </div>
        </div>
      </div>

      {detailSections.map((section, index) => (
        <DetailSection
          disabled={section.disabled}
          key={section.title}
          title={section.title}
          items={section.items}
          isFirst={index === 0}
        />
      ))}
    </>
  );
};
