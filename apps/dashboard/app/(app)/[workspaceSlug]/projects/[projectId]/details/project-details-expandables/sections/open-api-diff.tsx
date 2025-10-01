"use client";
import { shortenId } from "@/lib/shorten-id";
import { trpc } from "@/lib/trpc/client";
import { useLiveQuery } from "@tanstack/react-db";
import { ArrowRight } from "@unkey/icons";
import type { GetOpenApiDiffResponse } from "@unkey/proto";
import { InfoTooltip } from "@unkey/ui";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useProjectLayout } from "../../../layout-provider";
import { type DiffStatus, StatusIndicator } from "../../active-deployment-card/status-indicator";

const getDiffStatus = (data?: GetOpenApiDiffResponse): DiffStatus => {
  if (!data) {
    return "loading";
  }
  if (data.hasBreakingChanges) {
    return "breaking";
  }
  if (data.summary?.diff) {
    return "warning";
  }
  return "safe";
};

export const OpenApiDiff = () => {
  const params = useParams();
  const { collections, liveDeploymentId } = useProjectLayout();

  const query = useLiveQuery(
    (q) =>
      q
        .from({ deployment: collections.deployments })
        .orderBy(({ deployment }) => deployment.createdAt, "desc")
        .limit(2)
        .select((c) => ({
          id: c.deployment.id,
        })),
    [liveDeploymentId],
  );

  const [newDeployment, oldDeployment] = query.data ?? [];

  const diff = trpc.deploy.deployment.getOpenApiDiff.useQuery({
    newDeploymentId: newDeployment?.id ?? "",
    oldDeploymentId: oldDeployment?.id ?? "",
  });

  // @ts-expect-error I have no idea why this whines about type diff
  const status = getDiffStatus(diff.data);

  if (newDeployment && !oldDeployment) {
    return (
      <div className="rounded-[10px] flex items-center border border-gray-5 h-[52px] w-full max-w-md">
        <div className="bg-grayA-2 rounded-l-[10px] border-r border-grayA-3 h-full w-[52px] flex items-center justify-center shrink-0">
          <StatusIndicator status="safe" className="bg-transparent" />
        </div>
        <div className="flex flex-col flex-1 px-3 min-w-0">
          <div className="text-grayA-9 text-xs">current</div>
          <div className="text-accent-12 font-medium text-xs truncate">
            {shortenId(newDeployment.id)}
          </div>
        </div>
      </div>
    );
  }

  if (!newDeployment) {
    return null;
  }

  const diffUrl = `/${params?.workspaceSlug}/projects/${params?.projectId}/openapi-diff?from=${oldDeployment.id}&to=${newDeployment.id}`;
  return (
    <InfoTooltip
      content="View detailed API diff comparison"
      position={{ side: "top", align: "center" }}
      asChild
    >
      <Link href={diffUrl} className="hover:opacity-80 transition-opacity block">
        <div className="gap-4 items-center flex w-full">
          <div className="rounded-[10px] flex items-center border border-gray-5 h-[52px] w-full">
            <div className="bg-grayA-2 rounded-l-[10px] border-r border-grayA-3 h-full w-1/3 flex items-center justify-center">
              <StatusIndicator className="bg-transparent" />
            </div>
            <div className="flex flex-col flex-1 px-3">
              <div className="text-grayA-9 text-xs">from</div>
              <div className="text-accent-12 font-medium text-xs">
                {shortenId(oldDeployment.id)}
              </div>
            </div>
          </div>
          <ArrowRight className="shrink-0 text-gray-9 size-[14px]" size="sm-regular" />
          <div className="rounded-[10px] flex items-center border border-gray-5 h-[52px] w-full">
            <div className="bg-grayA-2 border-r border-grayA-3 h-full w-1/3 flex items-center justify-center">
              <StatusIndicator status={status} withSignal className="bg-transparent" />
            </div>
            <div className="flex flex-col flex-1 px-3">
              <div className="text-grayA-9 text-xs">to</div>
              <div className="text-accent-12 font-medium text-xs">
                {shortenId(newDeployment.id)}
              </div>
            </div>
          </div>
        </div>
      </Link>
    </InfoTooltip>
  );
};
