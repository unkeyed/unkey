import type { Deployment } from "@/lib/collections/deploy/deployments";
import { Button } from "@unkey/ui";
import Link from "next/link";
import { RedeployDialog } from "../../components/table/components/actions/redeploy-dialog";

/** Patterns matched against backend fault.Public messages to decide whether
 *  to show a "Go to Settings" link. Only errors fixable via project settings
 *  (dockerfile path, docker context, regions, git branch) belong here. */
const SETTINGS_HINT_PATTERNS = [
  // Dockerfile path / docker context (from build.go extractUserBuildError)
  "check that the file path is correct",
  "dockerfile appears to be empty",
  "build target stage was not found",
  "check the root directory",
  // Region configuration (from deploy_handler.go createTopologies)
  "configure at least one region",
  // Git branch (from deploy_handler.go buildImage)
  "git branch could not be resolved",
];

function isSettingsRelatedError(error: string): boolean {
  const lower = error.toLowerCase();
  return SETTINGS_HINT_PATTERNS.some((pattern) => lower.includes(pattern));
}

export type StepEntry = { error: string | null } | null | undefined;

export function FailedDeploymentBanner({
  steps,
  settingsUrl,
  onRedeploy,
  redeployOpen,
  onRedeployClose,
  deployment,
  instanceErrors,
}: {
  steps: StepEntry[];
  settingsUrl: string;
  onRedeploy: () => void;
  redeployOpen: boolean;
  onRedeployClose: () => void;
  deployment: Deployment;
  instanceErrors?: string[];
}) {
  const stepError = steps.find((s) => s?.error)?.error;
  const errorMessage = stepError
    ?? (instanceErrors && instanceErrors.length > 0
      ? instanceErrors.join("; ")
      : "Deployment failed");
  const showSettingsLink = isSettingsRelatedError(errorMessage);

  return (
    <div className="flex flex-col gap-3 animate-fade-slide-in">
      <div className="border border-errorA-4 bg-errorA-2 rounded-[14px] p-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="flex flex-col gap-0.5">
            <span className="text-sm font-medium text-error-11">Deployment failed</span>
            <span className="text-xs text-gray-11 max-w-150 break-after">
              {errorMessage}
              {showSettingsLink && (
                <>
                  {" "}
                  <Link
                    href={settingsUrl}
                    className="underline hover:text-gray-12 transition-colors"
                  >
                    Go to Settings
                  </Link>
                </>
              )}
            </span>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="primary" size="sm" onClick={onRedeploy} className="px-3">
            Redeploy
          </Button>
        </div>
      </div>
      <RedeployDialog
        isOpen={redeployOpen}
        onClose={onRedeployClose}
        selectedDeployment={deployment}
      />
    </div>
  );
}
