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
      gradientFromLeft: "bg-gradient-to-r from-success-2 via-success-2 to-transparent",
      gradientFromRight: "bg-gradient-to-l from-success-2 via-success-2 to-transparent",
    },
  },
  unstable: {
    label: "Unstable",
    icon: TriangleWarning2,
    message: "Intermittent health check failures. Metrics fluctuating significantly.",
    showBanner: true,
    colors: {
      dotBg: "bg-orange-9",
      dotRing: "hsl(var(--orangeA-4))",
      dotTextColor: "text-orangeA-7",
      bannerBg: "bg-orange-2",
      bannerBorder: "border-orangeA-3",
      textColor: "text-orange-12",
      gradientFromLeft: "bg-gradient-to-r from-orange-2 via-orange-2 to-transparent",
      gradientFromRight: "bg-gradient-to-l from-orange-2 via-orange-2 to-transparent",
    },
  },
  degraded: {
    label: "Degraded",
    icon: TriangleWarning2,
    message: "Performance degraded. Response times elevated.",
    showBanner: true,
    colors: {
      dotBg: "bg-warning-9",
      dotRing: "hsl(var(--warningA-4))",
      dotTextColor: "text-warningA-7",
      bannerBg: "bg-warning-2",
      bannerBorder: "border-warningA-3",
      textColor: "text-warning-12",
      gradientFromLeft: "bg-gradient-to-r from-warning-2 via-warning-2 to-transparent",
      gradientFromRight: "bg-gradient-to-l from-warning-2 via-warning-2 to-transparent",
    },
  },
  unhealthy: {
    label: "Unhealthy",
    icon: TriangleWarning2,
    message: "Critical health check failures detected. Immediate attention required.",
    showBanner: true,
    colors: {
      dotBg: "bg-error-9",
      dotRing: "hsl(var(--errorA-4))",
      dotTextColor: "text-errorA-7",
      bannerBg: "bg-error-2",
      bannerBorder: "border-errorA-3",
      textColor: "text-error-12",
      gradientFromLeft: "bg-gradient-to-r from-error-2 via-error-2 to-transparent",
      gradientFromRight: "bg-gradient-to-l from-error-2 via-error-2 to-transparent",
    },
  },
  recovering: {
    label: "Recovering",
    icon: TriangleWarning2,
    message: "System recovering from issues. Monitoring in progress.",
    showBanner: true,
    colors: {
      dotBg: "bg-feature-9",
      dotRing: "hsl(var(--featureA-4))",
      dotTextColor: "text-featureA-7",
      bannerBg: "bg-feature-2",
      bannerBorder: "border-featureA-3",
      textColor: "text-feature-12",
      gradientFromLeft: "bg-gradient-to-r from-feature-2 via-feature-2 to-transparent",
      gradientFromRight: "bg-gradient-to-l from-feature-2 via-feature-2 to-transparent",
    },
  },
  health_syncing: {
    label: "Syncing",
    icon: TriangleWarning2,
    message: "Health data synchronizing across regions.",
    showBanner: true,
    colors: {
      dotBg: "bg-info-9",
      dotRing: "hsl(var(--infoA-4))",
      dotTextColor: "text-infoA-7",
      bannerBg: "bg-info-2",
      bannerBorder: "border-infoA-2",
      textColor: "text-info-12",
      gradientFromLeft: "bg-gradient-to-r from-info-2 via-info-2 to-transparent",
      gradientFromRight: "bg-gradient-to-l from-info-2 via-info-2 to-transparent",
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
      gradientFromLeft: "bg-gradient-to-r from-gray-2 via-gray-2 to-transparent",
      gradientFromRight: "bg-gradient-to-l from-gray-2 via-gray-2 to-transparent",
    },
  },
  disabled: {
    label: "Disabled",
    icon: TriangleWarning2,
    message: "This instance has been disabled and is not serving traffic.",
    showBanner: true,
    colors: {
      dotBg: "bg-gray-9",
      dotRing: "hsl(var(--grayA-4))",
      dotTextColor: "text-grayA-7",
      bannerBg: "bg-gray-2",
      bannerBorder: "border-grayA-3",
      textColor: "text-gray-11",
      gradientFromLeft: "bg-gradient-to-r from-gray-2 via-gray-2 to-transparent",
      gradientFromRight: "bg-gradient-to-l from-gray-2 via-gray-2 to-transparent",
    },
  },
};

export { STATUS_CONFIG };
export type { StatusConfig, HealthStatus };
