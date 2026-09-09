import {
  AlertBanner,
  AlertBannerActions,
  AlertBannerDescription,
  AlertBannerTitle,
  Button,
} from "@unkey/ui";
import { IconTriangleWarningOutline18 } from "nucleo-ui-outline-18";

export default function AlertBannerVariants() {
  return (
    <div className="flex flex-col gap-3">
      <AlertBanner>
        <AlertBannerDescription>
          This project was imported from a Git repository.
        </AlertBannerDescription>
      </AlertBanner>
      <AlertBanner variant="warning">
        <AlertBannerTitle>Compute is paused</AlertBannerTitle>
        <AlertBannerDescription>
          Creating and deploying are paused without an active plan.
        </AlertBannerDescription>
        <AlertBannerActions>
          <Button variant="outline" size="md">
            Choose a plan
          </Button>
        </AlertBannerActions>
      </AlertBanner>
      <AlertBanner variant="error">
        <IconTriangleWarningOutline18 className="size-4" />
        <AlertBannerTitle>Deployment failed</AlertBannerTitle>
        <AlertBannerActions>
          <Button variant="outline" size="md">
            View logs
          </Button>
        </AlertBannerActions>
      </AlertBanner>
      <AlertBanner variant="error">
        <IconTriangleWarningOutline18 className="size-4" />
        <AlertBannerTitle>Deployment failed</AlertBannerTitle>
        <AlertBannerDescription>
          The build could not resolve the git branch for this project.
        </AlertBannerDescription>
        <AlertBannerActions>
          <Button variant="outline" size="md">
            Redeploy
          </Button>
        </AlertBannerActions>
      </AlertBanner>
    </div>
  );
}
