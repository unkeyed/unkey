"use client";

import { useProjectData } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(overview)/data-provider";
import { generateFakeDeployments } from "@/lib/fake-deployments";
import { shortenId } from "@/lib/shorten-id";
import { CodeBranch, CodeCommit } from "@unkey/icons";
import { TimestampInfo } from "@unkey/ui";
import { DeploymentStatusBadge } from "../../../../../components/deployment-status-badge";
import { Avatar } from "../../../../../components/git-avatar";

type Props = { deploymentId: string };

/**
 * Rendered when the user drills into a mock deployment (id prefix
 * `dep_fake_`). We can't reuse the real detail components here because
 * they read from collections (`deployments`, `domains`,
 * `sentinel-policies`) keyed on real IDs — and those collections won't
 * contain our mock entry. A simple metadata pane is enough for Dave to
 * feel the 3-panel layout during the prototype.
 */
export function FakeDetail({ deploymentId }: Props) {
  const { projectId } = useProjectData();
  const fakes = generateFakeDeployments(projectId);
  const deployment = fakes.find((f) => f.id === deploymentId);

  if (!deployment) {
    return (
      <div className="p-6 text-sm text-gray-11">
        Mock deployment not found for id {deploymentId}.
      </div>
    );
  }

  return (
    <div className="mx-auto flex max-w-[960px] flex-col gap-6 p-6">
      <header className="flex flex-col gap-2">
        <div className="flex items-center gap-3">
          <span className="font-mono text-[13px] font-semibold text-accent-12">
            {shortenId(deployment.id)}
          </span>
          <DeploymentStatusBadge status={deployment.status} />
          <span className="text-[11px] uppercase tracking-wide text-gray-11">Mock</span>
        </div>
        <h1 className="text-xl font-semibold text-accent-12">{deployment.gitCommitMessage}</h1>
      </header>

      <dl className="grid grid-cols-2 gap-4 text-[13px]">
        <Field label="Branch">
          <div className="flex items-center gap-1.5 font-mono">
            <CodeBranch iconSize="sm-regular" className="size-3.5 text-accent-11" />
            {deployment.gitBranch}
          </div>
        </Field>
        <Field label="Commit">
          <div className="flex items-center gap-1.5 font-mono">
            <CodeCommit iconSize="sm-regular" className="size-3.5 text-accent-11" />
            {deployment.gitCommitSha ?? "—"}
          </div>
        </Field>
        <Field label="Author">
          <div className="flex items-center gap-2">
            <Avatar
              src={deployment.gitCommitAuthorAvatarUrl}
              alt={deployment.gitCommitAuthorHandle ?? "Author"}
            />
            <span>{deployment.gitCommitAuthorHandle ?? "—"}</span>
          </div>
        </Field>
        <Field label="Deployed">
          <TimestampInfo
            value={deployment.createdAt}
            displayType="relative"
            className="text-[13px] text-accent-12"
          />
        </Field>
        <Field label="CPU">{deployment.cpuMillicores} mCPU</Field>
        <Field label="Memory">{deployment.memoryMib} MiB</Field>
      </dl>

      <p className="rounded-lg border border-dashed border-grayA-5 bg-gray-2 px-4 py-3 text-[12px] text-gray-11">
        This is a mock deployment rendered so the 3-panel layout has something to show for projects
        without real deploys. The full detail (network diagram, domains, build logs) lights up once
        a real deployment is selected.
      </p>
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <dt className="text-[11px] uppercase tracking-wide text-gray-11">{label}</dt>
      <dd className="text-accent-12">{children}</dd>
    </div>
  );
}
