"use client";

import { Plus } from "@unkey/icons";
import { Button } from "@unkey/ui";

export function CreateLogdrainButton({ onClick }: { onClick: () => void }) {
  return (
    <Button size="md" variant="primary" onClick={onClick}>
      <Plus iconSize="sm-medium" />
      Create Log Drain
    </Button>
  );
}
