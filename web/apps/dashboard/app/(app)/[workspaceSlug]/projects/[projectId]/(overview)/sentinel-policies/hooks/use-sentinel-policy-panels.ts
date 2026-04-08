"use client";

import type { SentinelPolicy } from "@/lib/collections/deploy/sentinel-policies.schema";
import { useCallback, useState } from "react";

export function useSentinelPolicyPanels() {
  const [isAddPanelOpen, setIsAddPanelOpen] = useState(false);
  const [editing, setEditing] = useState<SentinelPolicy | null>(null);

  const openAdd = useCallback(() => setIsAddPanelOpen(true), []);
  const closeAdd = useCallback(() => setIsAddPanelOpen(false), []);
  const openEdit = useCallback((policy: SentinelPolicy) => setEditing(policy), []);
  const closeEdit = useCallback(() => setEditing(null), []);

  return { isAddPanelOpen, openAdd, closeAdd, editing, openEdit, closeEdit };
}
