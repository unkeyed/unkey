"use client";

import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { useCallback, useState } from "react";
import { EditKeyAside } from "../builder/edit-key-aside";
import { RootKeysDataTable } from "./root-keys-data-table";

export function RootKeysListBuilder() {
  const [editingKeyId, setEditingKeyId] = useState<string | null>(null);
  const [isOpen, setIsOpen] = useState(false);

  const close = useCallback(() => setIsOpen(false), []);
  const forget = useCallback(() => setEditingKeyId(null), []);
  const edit = useCallback((rootKey: RootKey) => {
    setEditingKeyId(rootKey.id);
    setIsOpen(true);
  }, []);

  return (
    <>
      <RootKeysDataTable selectedKeyId={editingKeyId} onEditKey={edit} />
      {editingKeyId === null ? null : (
        <EditKeyAside
          key={editingKeyId}
          keyId={editingKeyId}
          isOpen={isOpen}
          onClose={close}
          onExitComplete={forget}
        />
      )}
    </>
  );
}
