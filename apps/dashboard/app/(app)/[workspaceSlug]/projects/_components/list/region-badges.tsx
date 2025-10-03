import { Earth } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";
import { RepoDisplay } from "./repo-display";

type RegionBadgesProps = {
  regions: string[];
  repository?: string;
};

export const RegionBadges = ({ regions, repository }: RegionBadgesProps) => {
  const visibleRegions = regions.slice(0, 1);
  const remainingRegions = regions.slice(1);
  const remainingCount = remainingRegions.length;

  return (
    <div className="flex gap-2 items-center mt-auto">
      {visibleRegions.map((region) => (
        <div
          key={region}
          className="bg-grayA-4 px-1.5 font-medium text-xs text-gray-12 rounded-full min-h-[22px] flex items-center gap-1.5"
        >
          <Earth iconsize="lg-medium" className="shrink-0" />
          {region}
        </div>
      ))}
      {remainingCount > 0 && (
        <InfoTooltip
          content={
            <div className="space-y-1">
              {remainingRegions.map((region) => (
                <div key={region} className="text-xs font-medium flex items-center gap-1.5">
                  <div className="w-1 h-1 bg-gray-8 rounded-full" />
                  {region}
                </div>
              ))}
            </div>
          }
        >
          <div className="bg-grayA-4 px-1.5 font-medium text-xs text-gray-12 rounded-full min-h-[22px] flex items-center">
            +{remainingCount} more
          </div>
        </InfoTooltip>
      )}
      {repository && (
        <RepoDisplay
          url={repository}
          className="bg-grayA-4 px-1.5 font-medium text-xs text-gray-12 rounded-full min-h-[22px] max-w-[130px]"
        />
      )}
    </div>
  );
};
