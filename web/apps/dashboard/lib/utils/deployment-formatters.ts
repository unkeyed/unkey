export function formatCpu(millicores: number): string {
  if (millicores === 0) {
    return "—";
  }
  // Fractional vCPU allocations (using 1000-based scale)
  if (millicores === 250) {
    return "1/4 vCPU";
  }
  if (millicores === 500) {
    return "1/2 vCPU";
  }
  if (millicores === 750) {
    return "3/4 vCPU";
  }
  if (millicores === 1000) {
    return "1 vCPU";
  }

  // Whole vCPUs
  if (millicores >= 1000 && millicores % 1000 === 0) {
    return `${millicores / 1000} vCPU`;
  }

  // Non-standard values - show as millicores
  return `${millicores}m vCPU`;
}

export function formatMemory(mib: number): string {
  if (mib === 0) {
    return "—";
  }
  // Convert to GiB when >= 1024 MiB
  if (mib >= 1024) {
    // Show decimals only if not a whole number
    return `${(mib / 1024).toFixed(mib % 1024 === 0 ? 0 : 1)} GiB`;
  }
  return `${mib} MiB`;
}
