import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { InfoTooltip } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";

type CriticalPermissionIndicatorProps = {
  rootKey: RootKey;
  isSelected: boolean;
};

export const CriticalPermissionIndicator = ({
  rootKey,
  isSelected,
}: CriticalPermissionIndicatorProps) => {
  const hasCriticalPerm = rootKey.permissionSummary.hasCriticalPerm;

  if (!hasCriticalPerm) {
    return <div className="w-2" />;
  }
  return (
    <InfoTooltip
      variant="inverted"
      content={
        <div className="text-xs">
          <div className="font-medium">
            This root key has critical permissions that can permanently destroy data or compromise
            security.
          </div>
        </div>
      }
    >
      <div
        className={cn(
          "size-2 rounded-full cursor-pointer ml-auto",
          isSelected ? "bg-orange-11" : "bg-orange-10 hover:bg-orange-11",
        )}
      />
    </InfoTooltip>
  );
};
