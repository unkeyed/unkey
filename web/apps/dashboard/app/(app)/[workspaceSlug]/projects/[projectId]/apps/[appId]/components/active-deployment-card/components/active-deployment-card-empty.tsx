import { IconPlusOutline18, IconSquareBulletListOutline18 } from "@unkey/icons";
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
  description = "Create a deployment to see it here.",
}: ActiveDeploymentCardEmptyProps) {
  return (
    <EmptySection
      title={title}
      description={description}
      icon={<IconSquareBulletListOutline18 />}
      className={cn("min-h-[200px]", className)}
    >
      {onCreateDeployment && (
        <Button variant="primary" onClick={onCreateDeployment} size="sm" className="mt-2">
          <IconPlusOutline18 className="size-4 mr-2" />
          Create deployment
        </Button>
      )}
    </EmptySection>
  );
}
