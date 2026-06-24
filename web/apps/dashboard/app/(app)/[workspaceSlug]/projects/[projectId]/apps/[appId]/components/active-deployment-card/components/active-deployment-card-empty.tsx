import { Cloud, Plus } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { EmptySection } from "../../../(overview)/components/empty-section";

type ActiveDeploymentCardEmptyProps = {
  onCreateDeployment?: () => void;
  className?: string;
  title?: string;
  description?: string;
};

export function ActiveDeploymentCardEmpty({
  onCreateDeployment,
  className,
  title = "No active deployments",
  description = "Your deployments will appear here once you push code to your connected repository or trigger a manual deployment.",
}: ActiveDeploymentCardEmptyProps) {
  return (
    <EmptySection
      title={title}
      description={description}
      icon={<Cloud className="size-6" />}
      className={cn("min-h-[200px]", className)}
    >
      {onCreateDeployment && (
        <Button onClick={onCreateDeployment} size="sm" className="mt-2">
          <Plus className="size-4 mr-2" />
          Create deployment
        </Button>
      )}
    </EmptySection>
  );
}
