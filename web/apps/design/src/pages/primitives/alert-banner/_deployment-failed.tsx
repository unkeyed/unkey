import {
  AlertBanner,
  AlertBannerActions,
  AlertBannerDescription,
  AlertBannerTitle,
  Button,
} from "@unkey/ui";

export function DeploymentFailedBanner() {
  return (
    <AlertBanner variant="error">
      <AlertBannerTitle>Deployment failed</AlertBannerTitle>
      <AlertBannerDescription className="max-w-150">
        The Dockerfile path is not correct. <a href="#settings">Go to Settings</a>
      </AlertBannerDescription>
      <AlertBannerActions>
        <Button variant="primary" size="sm" className="px-3">
          Redeploy
        </Button>
      </AlertBannerActions>
    </AlertBanner>
  );
}
