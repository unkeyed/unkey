import { IconCircleInfoOutline18 } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";
import { type Rejection, statusLabel } from "../policy-kinds/types";

export function Rejections({ rejects }: { rejects: readonly Rejection[] }) {
  return (
    <ul className="flex flex-col gap-0.5 text-xs leading-5 text-gray-11">
      {rejects.map(({ status, reason }) => (
        <li key={status}>
          <span className="font-medium text-gray-12">{statusLabel(status)}</span> {reason}
        </li>
      ))}
    </ul>
  );
}

export function RejectionTags({ rejects }: { rejects: readonly Rejection[] }) {
  return (
    <span className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs leading-5">
      {rejects.map(({ status, reason }) => (
        <span key={status} className="flex items-center gap-1 font-medium text-gray-12">
          {statusLabel(status)}
          <InfoTooltip content={reason} triggerClassName="flex text-gray-9 hover:text-gray-11">
            <IconCircleInfoOutline18 className="size-3" />
          </InfoTooltip>
        </span>
      ))}
    </span>
  );
}
