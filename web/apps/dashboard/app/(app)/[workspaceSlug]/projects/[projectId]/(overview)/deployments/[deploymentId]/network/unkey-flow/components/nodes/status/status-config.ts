import { CircleCheck, TriangleWarning2 } from "@unkey/icons";
import type { HealthStatus } from "../types";

type StatusColors = {
  dotBg: string;
  dotRing: string;
  dotTextColor: string;
  bannerBg: string;
  bannerBorder: string;
  textColor: string;
  gradientFromLeft: string;
  gradientFromRight: string;
};

type StatusConfig = {
  label: string;
  icon: typeof TriangleWarning2;
  message: string;
  colors: StatusColors;
  showBanner: boolean;
};

const STATUS_CONFIG: Record<HealthStatus, StatusConfig> = {
  normal: {
    label: "Normal",
    icon: CircleCheck,
    message: "",
    showBanner: false,
    colors: {
      dotBg: "bg-success-9",
      dotRing: "hsl(var(--successA-4))",
      dotTextColor: "text-gray-9",
      bannerBg: "bg-success-2",
      bannerBorder: "border-successA-2",
      textColor: "text-success-12",
      gradientFromLeft: "bg-linear-to-r from-success-2 via-success-2 to-transparent",
      gradientFromRight: "bg-linear-to-l from-success-2 via-success-2 to-transparent",
    },
  },
  unhealthy: {
    label: "Unhealthy",
    icon: TriangleWarning2,
    message: "Instance failed or sentinel health check failing.",
    showBanner: true,
    colors: {
      dotBg: "bg-error-9",
      dotRing: "hsl(var(--errorA-4))",
      dotTextColor: "text-errorA-7",
      bannerBg: "bg-error-2",
      bannerBorder: "border-errorA-3",
      textColor: "text-error-12",
      gradientFromLeft: "bg-linear-to-r from-error-2 via-error-2 to-transparent",
      gradientFromRight: "bg-linear-to-l from-error-2 via-error-2 to-transparent",
    },
  },
  health_syncing: {
    label: "Starting",
    icon: TriangleWarning2,
    message: "Instance is starting up.",
    showBanner: true,
    colors: {
      dotBg: "bg-info-9",
      dotRing: "hsl(var(--infoA-4))",
      dotTextColor: "text-infoA-7",
      bannerBg: "bg-info-2",
      bannerBorder: "border-infoA-2",
      textColor: "text-info-12",
      gradientFromLeft: "bg-linear-to-r from-info-2 via-info-2 to-transparent",
      gradientFromRight: "bg-linear-to-l from-info-2 via-info-2 to-transparent",
    },
  },
  unknown: {
    label: "Unknown",
    icon: TriangleWarning2,
    message: "Unable to determine health status. Investigating.",
    showBanner: true,
    colors: {
      dotBg: "bg-gray-9",
      dotRing: "hsl(var(--grayA-4))",
      dotTextColor: "text-grayA-7",
      bannerBg: "bg-gray-2",
      bannerBorder: "border-grayA-3",
      textColor: "text-gray-11",
      gradientFromLeft: "bg-linear-to-r from-gray-2 via-gray-2 to-transparent",
      gradientFromRight: "bg-linear-to-l from-gray-2 via-gray-2 to-transparent",
    },
  },
  disabled: {
    label: "Disabled",
    icon: TriangleWarning2,
    message: "Instance is inactive or sentinel is paused.",
    showBanner: true,
    colors: {
      dotBg: "bg-gray-9",
      dotRing: "hsl(var(--grayA-4))",
      dotTextColor: "text-grayA-7",
      bannerBg: "bg-gray-2",
      bannerBorder: "border-grayA-3",
      textColor: "text-gray-11",
      gradientFromLeft: "bg-linear-to-r from-gray-2 via-gray-2 to-transparent",
      gradientFromRight: "bg-linear-to-l from-gray-2 via-gray-2 to-transparent",
    },
  },
};

export { STATUS_CONFIG };
export type { StatusConfig, HealthStatus };
