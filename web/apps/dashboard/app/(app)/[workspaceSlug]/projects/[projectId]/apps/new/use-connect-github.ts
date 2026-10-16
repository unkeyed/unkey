"use client";

import { githubUrl } from "@/lib/github-url";
import { trpc } from "@/lib/trpc/client";
import { useMutation } from "@tanstack/react-query";
import { NEW_APP_MUTATION_KEY, useNewAppFlow } from "./flow";
import { newAppErrorMessage, useAppLifecycle } from "./use-app-lifecycle";

export const CONNECT_GITHUB = {
  title: "Connect GitHub",
  hint: "Install the Unkey GitHub app to import a repository",
};

export function useConnectGithub() {
  const { projectId, ensureApp, leave, locked } = useNewAppFlow();
  const { createGitPlaceholder } = useAppLifecycle(projectId);
  const prepareInstallation = trpc.github.prepareInstallation.useMutation();
  const install = useMutation({
    mutationKey: NEW_APP_MUTATION_KEY,
    mutationFn: async () => {
      const appId = await ensureApp("git", createGitPlaceholder);
      const { state } = await prepareInstallation.mutateAsync({ projectId, appId });
      return githubUrl.install(state);
    },
    onSuccess: (url) => {
      leave();
      window.location.href = url;
    },
  });

  return {
    connect: () => {
      if (!locked) {
        install.mutate();
      }
    },
    error: install.error
      ? `Could not start the GitHub install. ${newAppErrorMessage(install.error)}`
      : null,
  };
}
