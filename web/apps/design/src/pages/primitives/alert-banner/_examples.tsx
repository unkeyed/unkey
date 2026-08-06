import { TriangleWarning2 } from "@unkey/icons";
import {
  AlertBanner,
  AlertBannerActions,
  AlertBannerDescription,
  AlertBannerTitle,
  Button,
} from "@unkey/ui";

export function AlertBannerExamples() {
  return (
    <div className="flex w-full flex-col gap-3">
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
        <TriangleWarning2 iconSize="md-regular" />
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
