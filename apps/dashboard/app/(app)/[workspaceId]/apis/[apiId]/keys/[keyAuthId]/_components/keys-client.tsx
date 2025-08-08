"use client";

import { KeysListControlCloud } from "./components/control-cloud";
import { KeysListControls } from "./components/controls";
import { KeysList } from "./components/table/keys-list";

export const KeysClient = ({
  keyspaceId,
  apiId,
  workspaceId,
}: {
  keyspaceId: string;
  apiId: string;
  workspaceId: string;
}) => {
  return (
    <div className="flex flex-col">
      <KeysListControls keyspaceId={keyspaceId} />
      <KeysListControlCloud />
      <KeysList apiId={apiId} keyspaceId={keyspaceId} workspaceId={workspaceId} />
    </div>
  );
};
