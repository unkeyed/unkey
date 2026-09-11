"use client";

import { Plus } from "@unkey/icons";
import { Button } from "@unkey/ui";

export function CreateLogdrainButton({
  onClick,
  disabled,
}: {
  onClick: () => void;
  disabled?: boolean;
}) {
  return (
    <Button size="md" variant="primary" onClick={onClick} disabled={disabled}>
      <Plus iconSize="sm-medium" />
      Create Log Drain
    </Button>
  );
}
