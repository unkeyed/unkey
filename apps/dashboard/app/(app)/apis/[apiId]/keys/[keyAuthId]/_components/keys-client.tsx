"use client";

import { KeysListControlCloud } from "./components/control-cloud";
import { KeysListControls } from "./components/controls";
import { KeysList } from "./components/table/keys-list";

export const KeysClient = ({
  keyspaceId,
  apiId,
}: {
  keyspaceId: string;
  apiId: string;
}) => {
  return (
    <div className="flex flex-col">
      <KeysListControls keyspaceId={keyspaceId} />
      <KeysListControlCloud />
      <KeysList apiId={apiId} keyspaceId={keyspaceId} />
    </div>
  );
};
