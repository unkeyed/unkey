import {
  AlertBanner,
  AlertBannerActions,
  AlertBannerDescription,
  AlertBannerTitle,
  Button,
} from "@unkey/ui";

export function PlanGateBanner() {
  return (
    <AlertBanner variant="warning">
      <AlertBannerTitle>Compute is paused</AlertBannerTitle>
      <AlertBannerDescription>
        Creating and deploying are paused without an active plan.
      </AlertBannerDescription>
      <AlertBannerActions>
        <Button variant="outline" size="md" className="bg-background">
          Choose a plan
        </Button>
      </AlertBannerActions>
    </AlertBanner>
  );
}
