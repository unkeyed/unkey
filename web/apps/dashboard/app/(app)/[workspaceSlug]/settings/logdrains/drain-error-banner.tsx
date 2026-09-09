"use client";

import { AlertBanner, AlertBannerDescription, AlertBannerTitle } from "@unkey/ui";
import { IconTriangleWarningOutline18 } from "nucleo-ui-outline-18";
import type { DrainDetail } from "./drain-schema";

export function DrainErrorBanner({ status }: { status: DrainDetail["status"] }) {
  if (status !== "paused_by_failure") {
    return null;
  }

  return (
    <AlertBanner variant="error">
      <IconTriangleWarningOutline18 aria-hidden="true" />
      <AlertBannerTitle>Deliveries are failing</AlertBannerTitle>
      <AlertBannerDescription>
        Unkey paused this log drain after too many failed deliveries in a row. Fix the endpoint,
        then resume deliveries from the actions menu.
      </AlertBannerDescription>
    </AlertBanner>
  );
}
