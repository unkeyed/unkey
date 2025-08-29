import { trpc } from "@/lib/trpc/client";
import type { EnvVar } from "@/lib/trpc/routers/deploy/project/envs/getEnvs";
import { useCallback, useEffect, useState } from "react";

type UseEnvVarsProps = {
  projectId: string;
  environment: "production" | "preview" | "development";
};

export function useEnvVars({ environment, projectId }: UseEnvVarsProps) {
  const [editingId, setEditingId] = useState<string | null>(null);
  const [newVar, setNewVar] = useState({ key: "", value: "", isSecret: false });
  const [isAddingNew, setIsAddingNew] = useState(false);
  const trpcUtil = trpc.useUtils();

  const allEnvVars = trpcUtil.deploy.project.envs.getEnvs.getData({
    projectId,
  });

  const envVars = allEnvVars?.[environment] || [];
  const [localEnvVars, setLocalEnvVars] = useState<EnvVar[]>([]);

  // Sync server data with local state when it changes
  useEffect(() => {
    if (envVars.length > 0) {
      setLocalEnvVars(envVars);
    }
  }, [envVars]);

  const addVariable = useCallback(() => {
    if (!newVar.key.trim() || !newVar.value.trim()) {
      return;
    }

    const newEnvVar: EnvVar = {
      id: Date.now().toString(),
      key: newVar.key.trim(),
      value: newVar.value.trim(),
      isSecret: newVar.isSecret,
    };

    setLocalEnvVars((prev) => [...prev, newEnvVar]);
    setNewVar({ key: "", value: "", isSecret: false });
    setIsAddingNew(false);

    // TODO: Call create mutation when available
  }, [newVar]);

  const updateVariable = useCallback((id: string, updates: Partial<EnvVar>) => {
    setLocalEnvVars((prev) => prev.map((env) => (env.id === id ? { ...env, ...updates } : env)));
    setEditingId(null);

    // TODO: Call update mutation when available
  }, []);

  const deleteVariable = useCallback((id: string) => {
    setLocalEnvVars((prev) => prev.filter((env) => env.id !== id));

    // TODO: Call delete mutation when available
  }, []);

  const startEditing = useCallback((id: string) => {
    setEditingId(id);
    setIsAddingNew(false);
  }, []);

  const cancelEditing = useCallback(() => {
    setEditingId(null);
    setIsAddingNew(false);
    setNewVar({ key: "", value: "", isSecret: false });
  }, []);

  const startAdding = useCallback(() => {
    setIsAddingNew(true);
    setEditingId(null);
  }, []);

  return {
    envVars: localEnvVars,
    editingId,
    newVar,
    isAddingNew,
    addVariable,
    updateVariable,
    deleteVariable,
    startEditing,
    cancelEditing,
    startAdding,
    setNewVar,
  };
}
