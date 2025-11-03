import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { Book2, Cube, DoubleChevronRight } from "@unkey/icons";
import { Button, InfoTooltip } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useProject } from "../../layout-provider";
import { DetailSection } from "./detail-section";
import { createDetailSections } from "./sections";

type ProjectDetailsExpandableProps = {
  tableDistanceToTop: number;
  isOpen: boolean;
  onClose: () => void;
  projectId: string;
};

export const ProjectDetailsExpandable = ({
  tableDistanceToTop,
  isOpen,
  onClose,
  projectId,
}: ProjectDetailsExpandableProps) => {
  const { collections } = useProject();
  const query = useLiveQuery((q) =>
    q
      .from({ project: collection.projects })
      .where(({ project }) => eq(project.id, projectId))
      .join({ deployment: collections.deployments }, ({ deployment, project }) =>
        eq(deployment.id, project.liveDeploymentId),
      )
      .orderBy(({ project }) => project.id, "asc")
      .limit(1),
  );

  const data = query.data.at(0);
  const { data: domainsData } = useLiveQuery(
    (q) =>
      q
        .from({ domain: collections.domains })
        .where(({ domain }) => eq(domain.deploymentId, data?.project.liveDeploymentId))
        .select(({ domain }) => ({
          domain: domain.domain,
          environment: domain.sticky,
        }))
        .orderBy(({ domain }) => domain.id, "asc"),
    [data?.project.liveDeploymentId],
  );

  if (!data?.deployment) {
    return null;
  }

  const detailSections = createDetailSections({
    ...data.deployment,
    repository: data.project.gitRepositoryUrl,
  });

  // This "environment" domain never changes even when you do a rollback this one stays stable.
  const mainDomain = domainsData.find((d) => d.environment === "environment")?.domain;
  const gitShaAndBranchNameDomains = domainsData.filter((d) => d.environment !== "environment");

  return (
    <div className="flex">
      <div
        className={cn(
          "fixed right-0 bg-gray-1 border-l border-grayA-4 w-[360px] overflow-hidden z-50 pb-8",
          "transition-all duration-300 ease-out",
          "shadow-md",
          isOpen ? "translate-x-0 opacity-100" : "translate-x-full opacity-0",
        )}
        style={{
          top: `${tableDistanceToTop}px`,
          height: `calc(100vh - ${tableDistanceToTop}px)`,
          // Hardware acceleration
          willChange: isOpen ? "transform, opacity" : "auto",
        }}
      >
        {/* Scrollable content container */}
        <div className="h-full overflow-y-auto">
          {/* Details Header */}
          <div className="h-10 flex items-center justify-between border-b border-grayA-4 px-4 bg-gray-1 sticky top-0 z-10">
            <div className="items-center flex gap-2.5 pl-0.5 py-2">
              <Book2 iconSize="md-medium" />
              <span className="text-accent-12 font-medium text-sm">Details</span>
            </div>
            <InfoTooltip
              content="Hide details"
              asChild
              position={{
                side: "bottom",
                align: "end",
              }}
            >
              <Button variant="ghost" size="icon" onClick={onClose}>
                <DoubleChevronRight
                  iconSize="lg-medium"
                  className="text-gray-8 transition-transform duration-300 ease-out group-hover:text-gray-12"
                />
              </Button>
            </InfoTooltip>
          </div>

          {/* Animated content with stagger effect */}
          <div
            className={cn(
              "transition-all duration-500 ease-out",
              isOpen ? "translate-x-0 opacity-100" : "translate-x-6 opacity-0",
            )}
            style={{
              transitionDelay: isOpen ? "150ms" : "0ms",
            }}
          >
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
                  <span className="text-accent-12 font-medium text-sm truncate">
                    {data.project.name}
                  </span>
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
                            <div
                              key={d.domain}
                              className="text-xs font-medium flex items-center gap-1.5"
                            >
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
              <div
                key={section.title}
                className={cn(
                  "transition-all duration-300 ease-out",
                  isOpen ? "translate-x-0 opacity-100" : "translate-x-8 opacity-0",
                )}
                style={{
                  transitionDelay: isOpen ? `${200 + index * 50}ms` : "0ms",
                }}
              >
                <DetailSection title={section.title} items={section.items} isFirst={index === 0} />
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
};
