"use client";

import { platformPreviews } from "@/lib/flags/platform-previews";
import { useFlags } from "@/lib/flags/provider";
import { trpc } from "@/lib/trpc/client";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { useMutation } from "@tanstack/react-query";
import { toast } from "@unkey/ui";
import { useEffect } from "react";

export function usePlatformPreviews() {
  const utils = trpc.useUtils();
  const features = useFlags();
  useEffect(() => {
    if (!features.data || features.error) {
      return;
    }
    for (const preview of platformPreviews) {
      if (!features.data[preview.flag.slug]) {
        console.warn(`Preview flag "${preview.flag.slug}" is missing from the flags API response`);
      }
    }
  }, [features.data, features.error]);
  const previews = platformPreviews.flatMap((preview) => {
    const state = features.data?.[preview.flag.slug];
    if (!state) {
      return [];
    }
    return [
      {
        slug: preview.flag.slug,
        title: preview.title,
        description: preview.description,
        enabled: state.enabled,
        allowOptIn: state.allowOptIn,
        allowOptOut: state.allowOptOut,
      },
    ];
  });
  const update = useMutation({
    onMutate: () => features.cancel(),
    mutationFn: async (input: { slug: string; enabled: boolean }) => {
      await getUnkeyClient().workspace.updateFlag(input);
    },
    async onSuccess(_data, input) {
      await Promise.all([
        features.setCachedFlagValue(input),
        utils.workspace.getCurrent.invalidate(),
      ]);
      toast.success("Preview feature updated");
    },
    onError(error) {
      toast.error("We couldn't update the preview feature", {
        description: getErrorMessage(error),
      });
    },
  });

  return {
    previews,
    update,
    isLoading: features.isLoading,
    error: features.error,
    refetch: features.refetch,
  };
}
