"use client";

import { IconPlusOutline18 } from "@unkey/icons";
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
      <IconPlusOutline18 />
      Create Log Drain
    </Button>
  );
}
