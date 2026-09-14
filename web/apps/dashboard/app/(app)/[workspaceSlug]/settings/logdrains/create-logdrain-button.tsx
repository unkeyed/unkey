"use client";

import { IconPlusOutline18 } from "@unkey/icons";
import { Button } from "@unkey/ui";

export function CreateLogdrainButton({ onClick }: { onClick: () => void }) {
  return (
    <Button size="md" variant="primary" onClick={onClick}>
      <IconPlusOutline18 />
      Create Log Drain
    </Button>
  );
}
