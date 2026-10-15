"use client";

import type { EnvVar } from "@/lib/collections/deploy/env-vars";
import { IconEyeOutline12, IconEyeSlashOutline18 } from "@unkey/icons";
import { Button, CopyButton } from "@unkey/ui";
import { memo, useEffect, useRef, useState } from "react";

const AUTO_HIDE_MS = 10_000;

export const EnvVarValueCell = memo(function EnvVarValueCell({ envVar }: { envVar: EnvVar }) {
  const { value, type } = envVar;
  const [visible, setVisible] = useState(false);
  const hideTimeoutRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => () => clearTimeout(hideTimeoutRef.current), []);

  const startAutoHide = () => {
    clearTimeout(hideTimeoutRef.current);
    hideTimeoutRef.current = setTimeout(() => setVisible(false), AUTO_HIDE_MS);
  };

  const toggleReveal = () => {
    if (visible) {
      setVisible(false);
      clearTimeout(hideTimeoutRef.current);
      return;
    }
    setVisible(true);
    startAutoHide();
  };

  if (type === "writeonly") {
    return null;
  }

  return (
    <div data-sentry-mask className="flex min-w-0 items-center gap-1">
      <Button
        type="button"
        variant="ghost"
        size="icon"
        aria-label={visible ? "Hide value" : "Reveal value"}
        onClick={toggleReveal}
        className="relative shrink-0 p-0 text-gray-10 hover:bg-grayA-3 hover:text-gray-12 focus:ring-0 focus-visible:ring-1 [&_svg]:size-3"
      >
        {visible ? <IconEyeSlashOutline18 /> : <IconEyeOutline12 />}
      </Button>
      {visible ? (
        <>
          <span className="min-w-0 truncate rounded-md bg-grayA-3 px-1.5 py-0.5 font-mono text-xs text-gray-12">
            {value}
          </span>
          <CopyButton
            value={value}
            variant="ghost"
            onClick={startAutoHide}
            className="relative size-5 shrink-0 text-gray-10 [&_svg]:size-3"
          />
        </>
      ) : (
        <span className="font-mono text-xs tracking-widest text-gray-11">••••••••••••</span>
      )}
    </div>
  );
});
