"use client";

import { useId, useState } from "react";
import { useNewAppFlow } from "../flow";
import { PaneSubmit } from "./pane-actions";
import { VariableFields, useVariableDraft } from "./variables/variables-section";

export function VariablesPane({ appId }: { appId: string }) {
  const { projectId, dispatch } = useNewAppFlow();
  const draft = useVariableDraft(projectId, appId);
  const [saving, setSaving] = useState(false);
  const formId = useId();

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setSaving(true);
    const saved = await draft.save();
    setSaving(false);
    if (saved) {
      dispatch({ type: "next" });
    }
  };

  return (
    <form id={formId} className="flex flex-1 flex-col gap-3" onSubmit={submit}>
      <VariableFields draft={draft} />
      <PaneSubmit form={formId} loading={saving} disabled={saving || draft.environmentsLoading} />
    </form>
  );
}
