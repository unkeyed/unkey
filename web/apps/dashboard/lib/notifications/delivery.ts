import type { V2NotificationsTestDestinationResponseBodyData } from "@unkey/api/models/components";

export function deliveryPending(
  status: V2NotificationsTestDestinationResponseBodyData["status"],
): boolean {
  return status === "pending" || status === "sending";
}

export const deliveryLabels: Record<
  V2NotificationsTestDestinationResponseBodyData["status"],
  string
> = {
  pending: "Test queued",
  sending: "Sending test",
  accepted: "Provider accepted the test. Delivery is not confirmed.",
  delivered: "Test delivered",
  failed: "Test failed",
  suppressed: "Test suppressed",
  unknown: "Delivery outcome unknown. The test will not be resent automatically.",
};
