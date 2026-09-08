"use client";

import { TriangleWarning2 } from "@unkey/icons";
import { AlertBanner, AlertBannerDescription, AlertBannerTitle } from "@unkey/ui";
import type { DrainDetail } from "./drain-schema";

export function DrainErrorBanner({ status }: { status: DrainDetail["status"] }) {
  if (status !== "paused_by_failure") {
    return null;
  }

  return (
    <AlertBanner variant="error">
      <TriangleWarning2 iconSize="md-regular" aria-hidden="true" />
      <AlertBannerTitle>Deliveries are failing</AlertBannerTitle>
      <AlertBannerDescription>
        Unkey paused this log drain after too many failed deliveries in a row. Fix the endpoint,
        then resume deliveries from the actions menu.
      </AlertBannerDescription>
    </AlertBanner>
  );
}
