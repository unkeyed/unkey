import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { useMemo } from "react";
import {
  type ProcessedTimeseriesDataPoint,
  useFetchVerificationTimeseries,
} from "../bar-chart/use-fetch-timeseries";
import { STATUS_DEFINITIONS, type StatusInfo } from "./constants";

const RATE_LIMIT_THRESHOLD_PERCENT = 0.1; // 10%
const VALIDATION_ISSUE_THRESHOLD_PERCENT = 0.1; // 10%
const LOW_CREDITS_THRESHOLD_ABSOLUTE = 100;
const LOW_CREDITS_THRESHOLD_REFILL_PERCENT = 0.1; // 10%
const EXPIRY_THRESHOLD_HOURS = 24;

type AggregatedData = {
  total: number;
  error: number;
  rate_limited: number;
};

// Helper to aggregate timeseries data
const aggregateTimeseries = (timeseries: ProcessedTimeseriesDataPoint[]): AggregatedData => {
  return timeseries.reduce(
    (acc, point) => {
      acc.total += point.total;
      acc.error += point.error;
      acc.rate_limited += point.rate_limited ?? 0; // Handle potential undefined
      // Add other aggregations if necessary
      return acc;
    },
    { total: 0, error: 0, rate_limited: 0 },
  );
};

interface UseKeyStatusResult {
  primary: {
    label: string;
    color: string;
    icon: React.ReactNode;
  };
  count: number;
  tooltips: string[];
  isLoading: boolean;
  isError: boolean;
}

export const useKeyStatus = (keyAuthId: string, keyData: KeyDetails): UseKeyStatusResult => {
  const { timeseries, isError, isLoading } = useFetchVerificationTimeseries(keyAuthId, keyData.id);

  const statusResult = useMemo(() => {
    if (!keyData.enabled) {
      const disabledStatus = STATUS_DEFINITIONS.disabled;
      return {
        primary: {
          label: disabledStatus.label,
          color: disabledStatus.color,
          icon: disabledStatus.icon,
        },
        count: 0,
        tooltips: [disabledStatus.tooltip],
      };
    }

    const applicableStatuses: StatusInfo[] = [];
    const aggregatedData = aggregateTimeseries(timeseries);
    const totalVerifications = aggregatedData.total;

    if (
      totalVerifications > 0 &&
      aggregatedData.rate_limited / totalVerifications > RATE_LIMIT_THRESHOLD_PERCENT
    ) {
      applicableStatuses.push(STATUS_DEFINITIONS["rate-limited"]);
    }

    if (
      totalVerifications > 0 &&
      aggregatedData.error / totalVerifications > VALIDATION_ISSUE_THRESHOLD_PERCENT
    ) {
      applicableStatuses.push(STATUS_DEFINITIONS["validation-issues"]);
    }

    const remaining = keyData.key.remaining;
    const refillAmount = keyData.key.refillAmount;
    const isLowOnCredits =
      (remaining != null && remaining < LOW_CREDITS_THRESHOLD_ABSOLUTE) ||
      (refillAmount &&
        remaining != null &&
        refillAmount > 0 &&
        remaining < refillAmount * LOW_CREDITS_THRESHOLD_REFILL_PERCENT);

    if (isLowOnCredits) {
      applicableStatuses.push(STATUS_DEFINITIONS["low-credits"]);
    }

    // Check Expiry
    if (keyData.expires) {
      const hoursToExpiry = (keyData.expires * 1000 - Date.now()) / (1000 * 60 * 60);
      if (hoursToExpiry > 0 && hoursToExpiry <= EXPIRY_THRESHOLD_HOURS) {
        applicableStatuses.push(STATUS_DEFINITIONS["expires-soon"]);
      }
    }

    if (applicableStatuses.length === 0) {
      const operationalStatus = STATUS_DEFINITIONS.operational;
      return {
        primary: {
          label: operationalStatus.label,
          color: operationalStatus.color,
          icon: operationalStatus.icon,
        },
        count: 0,
        tooltips: [operationalStatus.tooltip],
      };
    }

    applicableStatuses.sort((a, b) => a.priority - b.priority);

    const primaryStatus = applicableStatuses[0];
    return {
      primary: {
        label: primaryStatus.label,
        color: primaryStatus.color,
        icon: primaryStatus.icon,
      },
      count: applicableStatuses.length - 1, // Don't count the primary one
      tooltips: applicableStatuses.map((s) => s.tooltip),
    };
  }, [keyData, timeseries]);

  return {
    ...statusResult,
    isLoading,
    isError,
  };
};
