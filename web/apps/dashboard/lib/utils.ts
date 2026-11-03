import type { TimeUnit } from "@unkey/ui";

export const isBrowser = typeof window !== "undefined";

export function debounce<T extends (...args: unknown[]) => unknown>(func: T, delay: number) {
  let timeoutId: ReturnType<typeof setTimeout>;

  function debounced(...args: Parameters<T>) {
    clearTimeout(timeoutId);
    timeoutId = setTimeout(() => {
      func(...args);
    }, delay);
  }

  debounced.cancel = () => {
    clearTimeout(timeoutId);
  };

  return debounced;
}

export const DEFAULT_LOGS_SINCE = "6h";

export const processTimeFilters = (date?: Date, newTime?: TimeUnit) => {
  if (date) {
    const hours = newTime?.HH ? Number.parseInt(newTime.HH) : 0;
    const minutes = newTime?.mm ? Number.parseInt(newTime.mm) : 0;
    const seconds = newTime?.ss ? Number.parseInt(newTime.ss) : 0;
    const newDate = new Date(date);
    newDate.setHours(hours, minutes, seconds, 0);
    return newDate;
  }
  const now = new Date();
  return now;
};

export function getBaseUrl() {
  if (typeof window !== "undefined") {
    // browser should use relative path
    return "";
  }

  // Vercel never exposes a custom domain assigned to a preview branch through
  // env vars, so deployments like the canary (app.unkey-canary.com on the
  // `canary` branch) need an explicit, branch-scoped override. Without it,
  // auth and billing redirects fall through to VERCEL_BRANCH_URL and strand
  // users on the *.vercel.app host.
  if (process.env.DASHBOARD_BASE_URL) {
    return process.env.DASHBOARD_BASE_URL;
  }

  // VERCEL_URL is always the auto-generated *.vercel.app deployment URL —
  // never the custom domain, even in production. The production custom
  // domain lives in VERCEL_PROJECT_PRODUCTION_URL.
  if (process.env.VERCEL_ENV === "production" && process.env.VERCEL_PROJECT_PRODUCTION_URL) {
    return `https://${process.env.VERCEL_PROJECT_PRODUCTION_URL}`;
  }

  if (process.env.VERCEL_BRANCH_URL) {
    // Stable branch URL for previews (e.g., your-app-git-branch-team.vercel.app)
    return `https://${process.env.VERCEL_BRANCH_URL}`;
  }

  if (process.env.VERCEL_URL) {
    // Fallback to deployment-specific URL if branch URL not available
    return `https://${process.env.VERCEL_URL}`;
  }

  // assume localhost
  return `http://localhost:${process.env.PORT ?? 3000}`;
}

type DeepMergeInput = Record<string, unknown>;

export const deepMerge = <T extends DeepMergeInput>(target: T, source: T): T => {
  const result = { ...target } as T;

  for (const key in source) {
    const sourceValue = source[key];
    const targetValue = result[key];

    if (
      sourceValue !== null &&
      typeof sourceValue === "object" &&
      !Array.isArray(sourceValue) &&
      targetValue !== null &&
      typeof targetValue === "object" &&
      !Array.isArray(targetValue)
    ) {
      result[key] = deepMerge(
        targetValue as DeepMergeInput,
        sourceValue as DeepMergeInput,
      ) as T[Extract<keyof T, string>];
    } else if (sourceValue !== undefined) {
      result[key] = sourceValue;
    }
  }

  return result;
};
