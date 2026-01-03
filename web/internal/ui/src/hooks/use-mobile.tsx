import * as React from "react";

type UseIsMobileOptions = {
  breakpoint?: number;
  defaultValue?: boolean;
};

export function useIsMobile(options: UseIsMobileOptions = {}): boolean {
  const { breakpoint = 768, defaultValue = false } = options;
  const [isMobile, setIsMobile] = React.useState<boolean>(defaultValue);

  React.useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    const mql = window.matchMedia(`(max-width: ${breakpoint - 1}px)`);
    const onChange = (event: MediaQueryListEvent) => {
      setIsMobile(event.matches);
    };
    mql.addEventListener("change", onChange);
    setIsMobile(mql.matches);
    return () => mql.removeEventListener("change", onChange);
  }, [breakpoint]);

  return isMobile;
}
