"use client";

import { useWorkspace } from "@/providers/workspace-provider";
import { type ReactNode, createContext, useContext, useMemo, useSyncExternalStore } from "react";
import type { Member } from "./notification-model";
import { type PreviewStore, SAMPLE_MEMBERS, previewStoreFor } from "./preview-store";

const PreviewStoreContext = createContext<PreviewStore | null>(null);

export function NotificationPreviewProvider({ children }: { children: ReactNode }) {
  const { workspace } = useWorkspace();
  if (!workspace) {
    return null;
  }
  return (
    <PreviewStoreContext.Provider key={workspace.id} value={previewStoreFor(workspace.id)}>
      {children}
    </PreviewStoreContext.Provider>
  );
}

export function useNotificationPreview() {
  const store = useContext(PreviewStoreContext);
  if (!store) {
    throw new Error("useNotificationPreview must be used within NotificationPreviewProvider");
  }
  const state = useSyncExternalStore(store.subscribe, store.getSnapshot, store.getServerSnapshot);
  const { user } = useWorkspace();

  const members = useMemo<Member[]>(() => {
    if (!user) {
      return [...SAMPLE_MEMBERS];
    }
    const you: Member = {
      userId: user.id,
      name: user.fullName ?? user.email,
      email: user.email,
      role: user.role === "admin" ? "admin" : "member",
      billingContact: false,
    };
    return [you, ...SAMPLE_MEMBERS];
  }, [user]);

  return {
    ...state,
    actions: store.actions,
    workspaceId: store.workspaceId,
    members,
    currentUserId: user?.id ?? null,
    canManage: user?.role === "admin",
  };
}
