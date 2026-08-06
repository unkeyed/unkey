import { CircleCheck, CircleInfo, CircleWarning, TriangleWarning2 } from "@unkey/icons";
import { AlertBanner, AlertBannerDescription } from "@unkey/ui";

export function AlertBannerVariants() {
  return (
    <div className="flex w-full flex-col gap-3">
      <AlertBanner variant="info">
        <CircleInfo iconSize="md-regular" />
        <AlertBannerDescription>
          Your workspace moves to the new billing period on 1 September.
        </AlertBannerDescription>
      </AlertBanner>
      <AlertBanner variant="success">
        <CircleCheck iconSize="md-regular" />
        <AlertBannerDescription>
          The domain is verified and now routes traffic.
        </AlertBannerDescription>
      </AlertBanner>
      <AlertBanner variant="warning">
        <TriangleWarning2 iconSize="md-regular" />
        <AlertBannerDescription>
          This key expires in 3 days. Rotate it to avoid downtime.
        </AlertBannerDescription>
      </AlertBanner>
      <AlertBanner variant="error">
        <CircleWarning iconSize="md-regular" />
        <AlertBannerDescription>
          The last deployment failed and the previous version is still live.
        </AlertBannerDescription>
      </AlertBanner>
    </div>
  );
}
