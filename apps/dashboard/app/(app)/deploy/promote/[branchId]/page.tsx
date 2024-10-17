import { getTenantId } from "@/lib/auth";
import { db } from "@/lib/db";
import { diffJson } from "diff";

import MergeConfirmationCard from "./merge";
export const revalidate = 0;

type Props = {
  params: {
    branchId: string;
  };
};

export default async function Page(props: Props) {
  const tenantId = getTenantId();

  const workspace = await db.query.workspaces.findFirst({
    where: (table, { and, eq, isNull }) =>
      and(eq(table.tenantId, tenantId), isNull(table.deletedAt)),
  });
  if (!workspace) {
    return <div>Workspace with tenantId: {tenantId} not found</div>;
  }

  const branch = await db.query.gatewayBranches.findFirst({
    where: (table, { eq, and }) =>
      and(eq(table.workspaceId, workspace.id), eq(table.id, props.params.branchId)),
    with: {
      parent: true,
    },
  });
  if (!branch) {
    return <div>No branch found</div>;
  }

  const diff = diffJson(
    JSON.parse(branch?.parent?.openapi ?? "{}"),
    JSON.parse(branch?.openapi ?? "{}"),
  );

  const hunks = diff.map((h) => {
    if (h.added) {
      return <pre className="bg-green-500/10 text-green-800">{h.value}</pre>;
    }
    if (h.removed) {
      return <pre className="text-red-800 bg-red-500/10">{h.value}</pre>;
    }

    const lines = h.value.split("\n");

    const display =
      lines.length <= 4
        ? lines
        : [lines.at(0), lines.at(1), "... <unchanged>", lines.at(-2), lines.at(-1)];

    return <pre className="text-gray-400">{display.join("\n")}</pre>;
  });

  return (
    <>
      <div className="flex justify-between gap-8">
        <div className="w-2/3">
          <code className="flex flex-col font-mono">{hunks}</code>
        </div>

        <MergeConfirmationCard
          sourceBranch={branch.name}
          targetBranch={branch.parent!.name}
          targetBranchId={branch.parent!.id}
        />
      </div>
    </>
  );
}
