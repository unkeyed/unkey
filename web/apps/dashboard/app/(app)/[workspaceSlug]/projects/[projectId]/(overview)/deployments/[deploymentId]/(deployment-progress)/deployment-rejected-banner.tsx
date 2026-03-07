import { CircleWarning } from "@unkey/icons";

export function DeploymentRejectedBanner() {
  return (
    <div className="border border-errorA-4 bg-errorA-2 rounded-[14px] p-5 flex flex-col gap-4">
      <div className="flex items-start gap-3">
        <div className="rounded-md bg-errorA-3 p-1.5 mt-0.5">
          <CircleWarning iconSize="md-regular" className="text-error-11" />
        </div>
        <div className="flex flex-col gap-1">
          <span className="text-sm font-medium text-error-11">Deployment Rejected</span>
          <span className="text-xs text-gray-11">
            This deployment was rejected by a project member and will not proceed.
          </span>
        </div>
      </div>
    </div>
  );
}
