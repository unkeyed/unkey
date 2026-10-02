import { type EffectCallback, useEffect } from "react";

export function useMountEffect(effect: EffectCallback): void {
  useEffect(effect, []);
}
