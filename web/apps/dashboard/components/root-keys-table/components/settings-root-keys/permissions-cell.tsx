import { cn } from "@/lib/utils";
import { InfoTooltip } from "@unkey/ui";
import { describePermissions } from "../../utils/describe-permission";

const MAX_VISIBLE = 2;

type PermissionsCellProps = {
  permissions: { name: string }[];
  isSelected: boolean;
};

export function PermissionsCell({ permissions, isSelected }: PermissionsCellProps) {
  const labels = describePermissions(permissions.map((permission) => permission.name));

  if (labels.length === 0) {
    return <span className="text-gray-8">—</span>;
  }

  const hidden = labels.slice(MAX_VISIBLE);

  return (
    <div className="flex min-w-0 items-center gap-2">
      <span
        className={cn(
          "truncate text-[13px] leading-5",
          isSelected ? "text-accent-12" : "text-gray-11",
        )}
      >
        {labels.slice(0, MAX_VISIBLE).join(", ")}
      </span>
      {hidden.length > 0 ? (
        <InfoTooltip
          className="p-0"
          content={
            <div className="flex max-h-[180px] max-w-xs flex-col gap-1 overflow-y-auto py-2">
              {hidden.map((label) => (
                <div key={label} className="px-3 text-xs text-gray-11">
                  {label}
                </div>
              ))}
            </div>
          }
          position={{ side: "top", align: "start", sideOffset: 5 }}
          asChild
        >
          <span className="shrink-0 whitespace-nowrap text-xs text-gray-9">+{hidden.length}</span>
        </InfoTooltip>
      ) : null}
    </div>
  );
}
