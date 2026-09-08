"use client";

import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { queryClient } from "@/lib/collections/client";
import { sanitizeImageRef, validateImageRef } from "@/lib/docker-image-ref";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import { ChevronLeft, Docker } from "@unkey/icons";
import { Button, Input, toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useId, useState } from "react";

type DeployImageCardProps = {
  projectId: string;
  appId: string;
  onBeforeNavigate?: () => void;
  expanded: boolean;
  onExpandedChange: (expanded: boolean) => void;
};

export const DeployImageCard = ({
  projectId,
  appId,
  onBeforeNavigate,
  expanded,
  onExpandedChange,
}: DeployImageCardProps) => {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const { gated, openPaywall, planGate } = useDeployActionGate();
  const [image, setImage] = useState("");
  const hintId = useId();

  const { data: environments } = trpc.deploy.environment.list.useQuery({ projectId });
  const appEnvironments = (environments ?? []).filter((e) => e.appId === appId);
  const environmentSlug =
    appEnvironments.find((e) => e.kind === "preview")?.slug ?? appEnvironments[0]?.slug;

  const createDeployment = useMutation({
    mutationFn: async (source: { environment: string; image: string }) => {
      const res = await getUnkeyClient().deployments.createDeployment({
        project: projectId,
        app: appId,
        environment: source.environment,
        image: { dockerImage: source.image },
      });
      return { deploymentId: res.data.deploymentId };
    },
    async onSuccess(data) {
      await queryClient.invalidateQueries({ queryKey: ["deployments", projectId] });
      onBeforeNavigate?.();
      router.push(
        routes.projects.apps.deployment({
          workspaceSlug: workspace.slug,
          projectId,
          appId,
          deploymentId: data.deploymentId,
        }),
      );
    },
    onError(error) {
      toast.error(getErrorMessage(error));
    },
  });

  const imageRef = sanitizeImageRef(image);
  const validation = validateImageRef(imageRef);
  const error = imageRef && !validation.ok ? validation.error : undefined;
  const warning = validation.ok ? validation.warning : undefined;
  const canDeploy = validation.ok && Boolean(environmentSlug) && !createDeployment.isLoading;

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    if (!canDeploy || !environmentSlug) {
      return;
    }
    if (gated) {
      openPaywall();
      return;
    }
    createDeployment.mutate({ environment: environmentSlug, image: imageRef });
  };

  return (
    <div className="border border-grayA-5 rounded-lg flex flex-col gap-4 py-[18px] px-4">
      <div className="flex justify-start items-center gap-4">
        <div className="size-8 rounded-[10px] grid place-items-center ring-1 ring-grayA-4 shadow-sm shadow-grayA-8/20 dark:shadow-none shrink-0">
          <Docker className="size-[18px] text-gray-12" iconSize="md-medium" />
        </div>
        <div className="flex flex-col gap-3">
          <span className="font-medium text-gray-12 text-[13px] leading-[9px]">
            Deploy an image
          </span>
          {expanded ? null : (
            <span className="text-gray-10 text-[13px] leading-[9px]">
              Pull an image from any public registry
            </span>
          )}
        </div>
        {expanded ? (
          <Button
            variant="ghost"
            className="ml-auto rounded-lg"
            onClick={() => onExpandedChange(false)}
          >
            <ChevronLeft className="size-[14px]! text-gray-12 shrink-0" />
            <span className="text-[13px] text-gray-12 font-medium">Back</span>
          </Button>
        ) : (
          <Button
            variant="outline"
            className="ml-auto rounded-lg border-grayA-4 hover:bg-grayA-2 shadow-sm hover:shadow-md transition-all"
            onClick={() => onExpandedChange(true)}
          >
            <Docker className="size-[18px]! text-gray-12 shrink-0" />
            <span className="text-[13px] text-gray-12 font-medium">Use a Docker image</span>
          </Button>
        )}
      </div>
      {expanded ? (
        <form
          onSubmit={handleSubmit}
          className="flex flex-col gap-2 animate-in fade-in slide-in-from-top-1 duration-200 ease-out motion-reduce:animate-none"
        >
          <div className="flex items-center gap-2">
            <Input
              value={image}
              onChange={(e) => setImage(e.target.value)}
              onBlur={() => setImage(imageRef)}
              onPaste={(e) => {
                const cleaned = sanitizeImageRef(e.clipboardData.getData("text"));
                e.preventDefault();
                setImage(cleaned);
              }}
              spellCheck={false}
              variant={error ? "error" : warning ? "warning" : "default"}
              aria-invalid={Boolean(error)}
              placeholder="ghcr.io/acme/mcp-server:v1.4.2"
              aria-label="Image reference"
              aria-describedby={hintId}
              className="h-9 bg-transparent border-grayA-4 font-mono text-xs flex-1 min-w-0"
              data-1p-ignore
            />
            <Button
              type="submit"
              variant="primary"
              size="lg"
              className="shrink-0"
              disabled={!canDeploy}
              loading={createDeployment.isLoading}
            >
              Deploy
            </Button>
          </div>
          <span id={hintId} className="text-gray-10 text-[13px]">
            Include a tag or digest, e.g. <span className="font-mono text-xs">:v1.4.2</span>
          </span>
          {(error ?? warning) ? (
            <output className={`text-[13px] ${error ? "text-error-11" : "text-warning-11"}`}>
              {error ?? warning}
            </output>
          ) : null}
        </form>
      ) : null}
      {planGate}
    </div>
  );
};
