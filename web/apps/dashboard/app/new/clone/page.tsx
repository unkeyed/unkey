import { getAuth } from "@/lib/auth/get-auth";
import { db } from "@/lib/db";
import { getRepository } from "@/lib/github";
import { Empty } from "@unkey/ui";
import Link from "next/link";
import { redirect } from "next/navigation";
import { DeployForm } from "./deploy-form";
import { InstallCta } from "./install-cta";
import { type CloneParams, buildCloneSearchString, parseCloneParams } from "./parse-params";

export const dynamic = "force-dynamic";

type Props = {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
};

export default async function CloneDeployPage(props: Props) {
  const search = await props.searchParams;

  const parsed = parseCloneParams(search);
  if (!parsed.ok) {
    return <InvalidParams message={parsed.error} />;
  }
  const params = parsed.params;
  const cloneSearch = buildCloneSearchString(params);
  const cloneHref = `/new/clone?${cloneSearch}`;

  const { userId, orgId } = await getAuth();
  if (!userId || !orgId) {
    redirect(`/auth/sign-in?redirect=${encodeURIComponent(cloneHref)}`);
  }

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) => and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
    columns: { id: true, slug: true },
    with: {
      githubAppInstallations: {
        columns: { installationId: true },
      },
    },
  });

  if (!workspace) {
    return <OnboardingRequired cloneHref={cloneHref} />;
  }

  const match = await findRepoInstallation(workspace.githubAppInstallations, params);

  if (!match) {
    return <InstallCta repository={params.repository} cloneSearch={cloneSearch} />;
  }

  return (
    <DeployForm
      workspaceSlug={workspace.slug}
      installationId={match.installationId}
      repository={{
        id: match.repo.id,
        fullName: match.repo.full_name,
        defaultBranch: match.repo.default_branch,
        htmlUrl: match.repo.html_url,
      }}
      params={params}
    />
  );
}

async function findRepoInstallation(
  installations: Array<{ installationId: number }>,
  params: CloneParams,
) {
  for (const installation of installations) {
    try {
      const repo = await getRepository(
        installation.installationId,
        params.repository.owner,
        params.repository.repo,
      );
      return { installationId: installation.installationId, repo };
    } catch (err) {
      // 404 means this installation can't see the repo; keep searching.
      // Any other error is an outage or auth issue the operator needs to see,
      // but we still fall through so the user gets the install CTA rather
      // than a 500 for a transient GitHub blip.
      const message = err instanceof Error ? err.message : String(err);
      if (!message.includes("GitHub API error 404")) {
        console.error("clone: getRepository failed", {
          installationId: installation.installationId,
          owner: params.repository.owner,
          repo: params.repository.repo,
          error: message,
        });
      }
    }
  }
  return null;
}

function InvalidParams({ message }: { message: string }) {
  return (
    <CenteredShell>
      <Empty>
        <Empty.Title>Invalid deploy link</Empty.Title>
        <Empty.Description>{message}</Empty.Description>
      </Empty>
    </CenteredShell>
  );
}

function OnboardingRequired({ cloneHref }: { cloneHref: string }) {
  return (
    <CenteredShell>
      <Empty>
        <Empty.Title>Finish setting up your workspace</Empty.Title>
        <Empty.Description>
          Create your workspace first, then open this link again to deploy.
        </Empty.Description>
        <Empty.Actions>
          <Link
            href="/new"
            className="rounded-lg border border-grayA-5 px-3 py-2 text-[13px] text-gray-12 hover:bg-grayA-2"
          >
            Start onboarding
          </Link>
          <Link
            href={cloneHref}
            className="ml-2 rounded-lg px-3 py-2 text-[13px] text-gray-10 hover:text-gray-12"
          >
            Reload deploy link
          </Link>
        </Empty.Actions>
      </Empty>
    </CenteredShell>
  );
}

function CenteredShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen flex items-center justify-center p-6">
      <div className="w-full max-w-md">{children}</div>
    </div>
  );
}
