export type FormattedParts = { value: string; unit: string };

export function formatCpuParts(millicores: number): FormattedParts {
  if (millicores === 0) {
    return { value: "—", unit: "" };
  }
  if (millicores === 250) {
    return { value: "1/4", unit: "vCPU" };
  }
  if (millicores === 500) {
    return { value: "1/2", unit: "vCPU" };
  }
  if (millicores === 750) {
    return { value: "3/4", unit: "vCPU" };
  }
  if (millicores === 1000) {
    return { value: "1", unit: "vCPU" };
  }
  if (millicores >= 1000 && millicores % 1000 === 0) {
    return { value: `${millicores / 1000}`, unit: "vCPU" };
  }
  return { value: `${millicores}m`, unit: "vCPU" };
}

export function formatMemoryParts(mib: number): FormattedParts {
  if (mib === 0) {
    return { value: "—", unit: "" };
  }
  if (mib >= 1024) {
    return { value: `${(mib / 1024).toFixed(mib % 1024 === 0 ? 0 : 1)}`, unit: "GiB" };
  }
  return { value: `${mib}`, unit: "MiB" };
}

export function formatStorageParts(mib: number): FormattedParts {
  if (mib === 0) {
    return { value: "None", unit: "" };
  }
  if (mib >= 1024) {
    return { value: `${(mib / 1024).toFixed(mib % 1024 === 0 ? 0 : 1)}`, unit: "GiB" };
  }
  return { value: `${mib}`, unit: "MiB" };
}

// Pick a binary unit (B/s, KiB/s, MiB/s, GiB/s) based on the magnitude.
// Idle pods read as "—" (the dash display value) instead of "0 B/s" so the chart row header stays
// honest about the absence of data versus a true zero rate.
export function formatBytesPerSecondParts(bytesPerSec: number): FormattedParts {
  if (!Number.isFinite(bytesPerSec) || bytesPerSec <= 0) {
    return { value: "—", unit: "" };
  }
  const KiB = 1024;
  const MiB = 1024 * KiB;
  const GiB = 1024 * MiB;
  if (bytesPerSec >= GiB) {
    return { value: (bytesPerSec / GiB).toFixed(2), unit: "GiB/s" };
  }
  if (bytesPerSec >= MiB) {
    return { value: (bytesPerSec / MiB).toFixed(2), unit: "MiB/s" };
  }
  if (bytesPerSec >= KiB) {
    return { value: (bytesPerSec / KiB).toFixed(1), unit: "KiB/s" };
  }
  return { value: `${Math.round(bytesPerSec)}`, unit: "B/s" };
}

// Cumulative network bytes over a window. Same auto-scaling shape as the
// per-second variant but without the "/s" suffix. Used for "total sent"
// / "total received" labels next to the peak-rate headline.
export function formatNetworkBytesParts(bytes: number): FormattedParts {
  // Negative or non-finite is "no data". Zero is real data — a window with
  // genuinely zero traffic should render "0 B", not look like the chart is
  // missing.
  if (!Number.isFinite(bytes) || bytes < 0) {
    return { value: "—", unit: "" };
  }
  if (bytes === 0) {
    return { value: "0", unit: "B" };
  }
  const KiB = 1024;
  const MiB = 1024 * KiB;
  const GiB = 1024 * MiB;
  if (bytes >= GiB) {
    return { value: (bytes / GiB).toFixed(2), unit: "GiB" };
  }
  if (bytes >= MiB) {
    return { value: (bytes / MiB).toFixed(2), unit: "MiB" };
  }
  if (bytes >= KiB) {
    return { value: (bytes / KiB).toFixed(1), unit: "KiB" };
  }
  return { value: `${Math.round(bytes)}`, unit: "B" };
}
