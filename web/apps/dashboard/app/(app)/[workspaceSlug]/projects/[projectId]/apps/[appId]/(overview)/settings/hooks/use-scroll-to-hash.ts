import { useEffect } from "react";

// Settings sections render after async data loads, so the browser's native
// hash scroll fires before the target exists, and content loading in above it
// shifts its position afterwards. Re-pin the target to the top across a short
// window so it stays put while the layout settles, bailing if the user scrolls.
export function useScrollToHash() {
  useEffect(() => {
    const id = window.location.hash.slice(1);
    if (!id) {
      return;
    }

    let cancelled = false;
    let timer: ReturnType<typeof setTimeout>;
    let attempts = 0;

    const cancel = () => {
      cancelled = true;
    };
    window.addEventListener("wheel", cancel, { passive: true });
    window.addEventListener("touchmove", cancel, { passive: true });
    window.addEventListener("keydown", cancel);

    const pin = () => {
      if (cancelled) {
        return;
      }
      document.getElementById(id)?.scrollIntoView({ block: "start" });
      if (attempts++ < 25) {
        timer = setTimeout(pin, 80);
      }
    };
    pin();

    return () => {
      cancelled = true;
      clearTimeout(timer);
      window.removeEventListener("wheel", cancel);
      window.removeEventListener("touchmove", cancel);
      window.removeEventListener("keydown", cancel);
    };
  }, []);
}
