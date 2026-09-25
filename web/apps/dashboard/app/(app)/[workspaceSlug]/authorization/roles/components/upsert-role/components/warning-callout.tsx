import { formatNumber } from "@/lib/fmt";
import { IconTriangleWarningOutline18 } from "@unkey/icons";
import { AlertBanner, AlertBannerDescription, InlineLink } from "@unkey/ui";

interface RoleWarningCalloutProps {
  count: number;
  type: "keys" | "permissions";
}

export const RoleWarningCallout = ({ count, type }: RoleWarningCalloutProps) => {
  const itemText = type === "keys" ? "keys" : "permissions";
  const settingsText = type === "keys" ? "key settings" : "permission settings";

  return (
    <AlertBanner variant="default" className="[&>svg]:text-warning-9">
      <IconTriangleWarningOutline18 aria-hidden="true" />
      <AlertBannerDescription>
        <span className="font-medium">Warning:</span> This role has {formatNumber(count)} {itemText}{" "}
        assigned. Use the{" "}
        <InlineLink
          className="underline"
          target="_blank"
          rel="noopener noreferrer"
          href="https://www.unkey.com/docs/api-reference/overview"
          label="API"
        />{" "}
        or {settingsText} to manage these assignments.
      </AlertBannerDescription>
    </AlertBanner>
  );
};
