"use client";

import { useState } from "react";
import { type DestinationEditorTarget, DestinationPanel } from "./destination-panel";
import { useNotifications } from "./notification-context";
import { useResourceNames } from "./notification-ui";
import { type RuleEditorTarget, RulePanel } from "./rule-panel";

type EditorState<Target> = { session: number; isOpen: boolean; target: Target };

function useEditorState<Target>() {
  const [state, setState] = useState<EditorState<Target> | null>(null);
  return {
    state,
    open: (target: Target) =>
      setState((previous) => ({ session: (previous?.session ?? 0) + 1, isOpen: true, target })),
    close: () => setState((previous) => (previous ? { ...previous, isOpen: false } : previous)),
  };
}

export function useRuleEditor() {
  const notifications = useNotifications();
  const resources = useResourceNames();
  const editor = useEditorState<RuleEditorTarget>();

  const element = editor.state ? (
    <RulePanel
      key={`rule-${editor.state.session}`}
      target={editor.state.target}
      isOpen={editor.state.isOpen}
      onClose={editor.close}
      members={notifications.members}
      destinations={notifications.destinations}
      resources={resources}
      currentUserId={notifications.currentUserId}
    />
  ) : null;

  return { open: editor.open, element };
}

export function useDestinationEditor() {
  const notifications = useNotifications();
  const editor = useEditorState<DestinationEditorTarget>();

  const element = editor.state ? (
    <DestinationPanel
      key={`destination-${editor.state.session}`}
      target={editor.state.target}
      isOpen={editor.state.isOpen}
      onClose={editor.close}
      rules={notifications.rules}
    />
  ) : null;

  return { open: editor.open, element };
}
