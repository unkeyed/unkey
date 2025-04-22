"use client";

import { useEffect, useState } from "react";

type DeviceType = "mobile" | "tablet" | "desktop";

const BREAKPOINTS = {
  mobile: 768,
  tablet: 1024,
};

export function useResponsive() {
  const [device, setDevice] = useState<DeviceType>(() => {
    if (typeof window === "undefined") {
      return "desktop";
    }
    const w = window.innerWidth;
    if (w <= BREAKPOINTS.mobile) {
      return "mobile";
    }
    if (w <= BREAKPOINTS.tablet) {
      return "tablet";
    }
    return "desktop";
  });

  useEffect(() => {
    const onResize = () => {
      const w = window.innerWidth;
      const newDevice: DeviceType =
        w <= BREAKPOINTS.mobile ? "mobile" : w <= BREAKPOINTS.tablet ? "tablet" : "desktop";
      setDevice(newDevice);
    };

    window.addEventListener("resize", onResize);
    onResize();

    return () => {
      window.removeEventListener("resize", onResize);
    };
  }, []);

  return {
    isMobile: device === "mobile",
    isTablet: device === "tablet",
    isDesktop: device === "desktop",
    device,
  };
}
