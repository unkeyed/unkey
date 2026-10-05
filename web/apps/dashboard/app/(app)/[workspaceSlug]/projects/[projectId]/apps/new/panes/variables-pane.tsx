"use client";

import { Button } from "@unkey/ui";
import { useId, useState } from "react";
import { useNewAppFlow } from "../flow";
import { PaneActions } from "./pane-actions";
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
      <PaneActions>
        <Button
          type="submit"
          form={formId}
          variant="primary"
          size="sm"
          className="px-3"
          loading={saving}
          disabled={saving || draft.environmentsLoading}
        >
          Continue
        </Button>
      </PaneActions>
    </form>
  );
}
