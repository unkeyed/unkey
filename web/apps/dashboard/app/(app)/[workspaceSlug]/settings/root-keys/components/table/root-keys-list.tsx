"use client";

import { useCallback, useState } from "react";
import { EditKeyAside } from "../builder/edit-key-aside";
import { useRootKeysV2List } from "./hooks/use-root-keys-v2-list";
import { RootKeysDataTable } from "./root-keys-data-table";
import type { RootKey } from "./root-keys-v2";

export function RootKeysList({ onCreate }: { onCreate: () => void }) {
  const list = useRootKeysV2List();
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
      <RootKeysDataTable
        selectedKeyId={editingKeyId}
        onEditKey={edit}
        onCreate={onCreate}
        list={list}
      />
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
