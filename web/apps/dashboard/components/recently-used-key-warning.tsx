import { formatTimeSinceLastUse } from "@/lib/recently-used-key";
import { IconTriangleWarningOutline12 } from "@unkey/icons";
import { AlertBanner, AlertBannerDescription, AlertBannerTitle } from "@unkey/ui";

export const RecentlyUsedKeyWarning = ({ lastUsedAt }: { lastUsedAt: number }) => {
  return (
    <AlertBanner variant="warning" className="mt-2">
      <IconTriangleWarningOutline12 className="size-3.5" aria-hidden="true" />
      <AlertBannerTitle>Still in use</AlertBannerTitle>
      <AlertBannerDescription>
        This key was last used {formatTimeSinceLastUse(lastUsedAt)}. Deleting it will immediately
        break anything that is still authenticating with it.
      </AlertBannerDescription>
    </AlertBanner>
  );
};
