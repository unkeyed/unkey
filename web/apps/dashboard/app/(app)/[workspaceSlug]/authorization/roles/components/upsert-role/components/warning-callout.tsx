import { formatNumber } from "@/lib/fmt";
import { TriangleWarning } from "@unkey/icons";
import { InlineLink } from "@unkey/ui";

interface RoleWarningCalloutProps {
  count: number;
  type: "keys" | "permissions";
}

export const RoleWarningCallout = ({ count, type }: RoleWarningCalloutProps) => {
  const itemText = type === "keys" ? "keys" : "permissions";
  const settingsText = type === "keys" ? "key settings" : "permission settings";

  return (
    <div className="rounded-xl bg-grayA-3 dark:bg-black border border-grayA-3 flex items-center gap-4 px-[22px] py-6">
      <div className="bg-gray-4 size-8 rounded-full flex items-center justify-center flex-shrink-0">
        <TriangleWarning className="text-warning-9" iconSize="xl-medium" />
      </div>
      <div className="text-gray-12 text-[13px] leading-6">
        <span className="font-medium">Warning:</span> This role has {formatNumber(count)} {itemText}{" "}
        assigned. Use the{" "}
        <InlineLink
          className="underline"
          target="_blank"
          rel="noopener noreferrer"
          href="https://www.unkey.com/docs/api-reference/v2/overview"
          label="API"
        />{" "}
        or {settingsText} to manage these assignments.
      </div>
    </div>
  );
};
