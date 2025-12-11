import * as React from "react";

type UseIsMobileOptions = {
  breakpoint?: number;
  defaultValue?: boolean;
};

export function useIsMobile(options: UseIsMobileOptions = {}): boolean | undefined {
  const { breakpoint = 768, defaultValue } = options;
  const [isMobile, setIsMobile] = React.useState<boolean | undefined>(defaultValue);

  React.useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    const mql = window.matchMedia(`(max-width: ${breakpoint - 1}px)`);
    const onChange = () => {
      setIsMobile(window.innerWidth < breakpoint);
    };
    mql.addEventListener("change", onChange);
    setIsMobile(window.innerWidth < breakpoint);
    return () => mql.removeEventListener("change", onChange);
  }, [breakpoint]);

  return isMobile;
}
