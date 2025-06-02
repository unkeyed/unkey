import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { useMemo } from "react";
import {
  type ProcessedTimeseriesDataPoint,
  useFetchVerificationTimeseries,
} from "../bar-chart/use-fetch-timeseries";
import { STATUS_DEFINITIONS, type StatusInfo } from "./constants";

const RATE_LIMIT_THRESHOLD_PERCENT = 0.1; // 10%
const VALIDATION_ISSUE_THRESHOLD_PERCENT = 0.1; // 10%
const LOW_CREDITS_THRESHOLD_ABSOLUTE = 0;
const LOW_CREDITS_THRESHOLD_REFILL_PERCENT = 0.1; // 10%
const EXPIRY_THRESHOLD_HOURS = 24;

type AggregatedData = {
  total: number;
  error: number;
  rate_limited: number;
};

const aggregateTimeseries = (timeseries: ProcessedTimeseriesDataPoint[]): AggregatedData => {
  return timeseries.reduce(
    (acc, point) => {
      acc.total += point.total;
      acc.error += point.error;
      acc.rate_limited += point.rate_limited ?? 0;
      return acc;
    },
    { total: 0, error: 0, rate_limited: 0 },
  );
};

type UseKeyStatusResult = {
  primary: {
    label: string;
    color: string;
    icon: React.ReactNode;
  };
  count: number;
  statuses: StatusInfo[];
  isLoading: boolean;
  isError: boolean;
};

const LOADING_PRIMARY = {
  label: "Loading",
  color: "bg-grayA-3",
  icon: null,
};

export const useKeyStatus = (keyAuthId: string, keyData: KeyDetails): UseKeyStatusResult => {
  const { timeseries, isError, isLoading } = useFetchVerificationTimeseries(keyAuthId, keyData.id);

  const statusResult = useMemo(() => {
    // Handle case where keyData might not be loaded yet
    if (!keyData) {
      return {
        primary: LOADING_PRIMARY,
        count: 0,
        statuses: [],
      };
    }

    if (isLoading && timeseries.length === 0) {
      return {
        primary: LOADING_PRIMARY,
        count: 0,
        statuses: [],
      };
    }

    if (isError) {
      const fallbackStatus = keyData.enabled
        ? STATUS_DEFINITIONS.operational
        : STATUS_DEFINITIONS.disabled;
      return {
        primary: {
          label: fallbackStatus.label,
          color: fallbackStatus.color,
          icon: fallbackStatus.icon,
        },
        count: 0,
        statuses: [fallbackStatus],
      };
    }

    if (!keyData.enabled) {
      const disabledStatus = STATUS_DEFINITIONS.disabled;
      return {
        primary: {
          label: disabledStatus.label,
          color: disabledStatus.color,
          icon: disabledStatus.icon,
        },
        count: 0,
        statuses: [disabledStatus],
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

    const remaining = keyData.key.credits.remaining;
    const refillAmount = keyData.key.credits.refillAmount;
    const isLowOnCredits =
      (remaining != null && remaining === LOW_CREDITS_THRESHOLD_ABSOLUTE) ||
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
      // Ensure current time used is consistent if needed, Date.now() is fine here
      if (hoursToExpiry > 0 && hoursToExpiry <= EXPIRY_THRESHOLD_HOURS) {
        applicableStatuses.push(STATUS_DEFINITIONS["expires-soon"]);
      }
    }

    // Handle Operational state (if no issues found)
    if (applicableStatuses.length === 0) {
      const operationalStatus = STATUS_DEFINITIONS.operational;
      return {
        primary: {
          label: operationalStatus.label,
          color: operationalStatus.color,
          icon: operationalStatus.icon,
        },
        count: 0,
        statuses: [operationalStatus], // Return array with the single operational status
      };
    }

    applicableStatuses.sort((a, b) => a.priority - b.priority); // Sort by priority

    const primaryStatus = applicableStatuses[0]; // Highest priority is the first element
    return {
      primary: {
        label: primaryStatus.label,
        color: primaryStatus.color,
        icon: primaryStatus.icon,
      },
      count: applicableStatuses.length - 1, // Count of *other* statuses besides primary
      statuses: applicableStatuses, // Return the full sorted array of applicable statuses
    };
  }, [keyData, timeseries, isLoading, isError]);

  return {
    ...statusResult,
    isLoading: isLoading || !keyData,
    isError,
  };
};
