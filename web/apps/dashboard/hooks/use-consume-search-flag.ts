"use client";

import { parseAsBoolean, useQueryState } from "nuqs";
import { useEffect, useRef } from "react";

export function useConsumeSearchFlag(
  key: string,
  settled: boolean,
  onConsume: (() => void) | null,
) {
  const [flag, setFlag] = useQueryState(key, parseAsBoolean.withOptions({ history: "replace" }));
  const consumed = useRef(false);

  useEffect(() => {
    if (!flag || !settled || consumed.current) {
      return;
    }
    consumed.current = true;
    void setFlag(null);
    onConsume?.();
  }, [flag, settled, onConsume, setFlag]);
}
