import { useCallback, useState } from "react";

export type EnvVar = {
  id: string;
  key: string;
  value: string;
  isSecret?: boolean;
};

type UseEnvironmentVariablesProps = {
  initialVars: EnvVar[];
};

export function useEnvVars({ initialVars }: UseEnvironmentVariablesProps) {
  const [envVars, setEnvVars] = useState<EnvVar[]>(initialVars);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [newVar, setNewVar] = useState({ key: "", value: "", isSecret: false });
  const [isAddingNew, setIsAddingNew] = useState(false);

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

    setEnvVars((prev) => [...prev, newEnvVar]);
    setNewVar({ key: "", value: "", isSecret: false });
    setIsAddingNew(false);
  }, [newVar]);

  const updateVariable = useCallback((id: string, updates: Partial<EnvVar>) => {
    setEnvVars((prev) => prev.map((env) => (env.id === id ? { ...env, ...updates } : env)));
    setEditingId(null);
  }, []);

  const deleteVariable = useCallback((id: string) => {
    setEnvVars((prev) => prev.filter((env) => env.id !== id));
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
    envVars,
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
