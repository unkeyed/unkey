import { IconTriangleWarningOutline18 } from "@unkey/icons";
import {
  AlertBanner,
  AlertBannerActions,
  AlertBannerDescription,
  AlertBannerMeta,
  AlertBannerTitle,
} from "@unkey/ui";

export default function AlertBannerCard() {
  return (
    <div className="flex flex-col gap-3">
      <AlertBanner variant="warning" appearance="card">
        <IconTriangleWarningOutline18 className="size-4" />
        <AlertBannerTitle>Out of memory</AlertBannerTitle>
        <AlertBannerMeta>5 restarts · 54 minutes ago</AlertBannerMeta>
        <AlertBannerActions>
          <a href="#logs" className="font-medium text-gray-11 text-xs hover:text-gray-12">
            View logs
          </a>
        </AlertBannerActions>
        <AlertBannerDescription>
          Your app hit its memory limit. Fix a leak or raise the limit.
        </AlertBannerDescription>
      </AlertBanner>
      <AlertBanner variant="error" appearance="card">
        <IconTriangleWarningOutline18 className="size-4" />
        <AlertBannerTitle>Crashing</AlertBannerTitle>
        <AlertBannerMeta>9 restarts · 2 minutes ago</AlertBannerMeta>
        <AlertBannerDescription>
          Your app exits right after it starts. Check your logs.
        </AlertBannerDescription>
      </AlertBanner>
    </div>
  );
}
