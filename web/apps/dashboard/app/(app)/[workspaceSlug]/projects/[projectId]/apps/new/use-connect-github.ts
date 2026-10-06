"use client";

import { githubInstallUrl } from "@/lib/github-url";
import { trpc } from "@/lib/trpc/client";
import { getErrorMessage } from "@/lib/unkey-client";
import { useState } from "react";
import { useNewAppFlow } from "./flow";
import { useAppLifecycle } from "./use-app-lifecycle";

export function useConnectGithub() {
  const { projectId, ensureApp } = useNewAppFlow();
  const { createGitPlaceholder } = useAppLifecycle(projectId);
  const prepareInstallation = trpc.github.prepareInstallation.useMutation();
  const [connecting, setConnecting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const connect = async () => {
    setConnecting(true);
    setError(null);
    try {
      const app = await ensureApp("git", createGitPlaceholder);
      if (!app.ok) {
        setConnecting(false);
        setError(app.error);
        return;
      }
      const [{ state }] = await Promise.all([
        prepareInstallation.mutateAsync({ projectId, appId: app.appId }),
        app.applyDefaults(),
      ]);
      window.location.href = githubInstallUrl(state);
    } catch (error) {
      setConnecting(false);
      setError(`Could not start the GitHub install. ${getErrorMessage(error)}`);
    }
  };

  return { connect, connecting, error };
}
