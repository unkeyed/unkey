export function formatCpu(millicores: number): string {
  if (millicores === 0) {
    return "—";
  }
  if (millicores === 256) {
    return "1/4 vCPU";
  }
  if (millicores === 512) {
    return "1/2 vCPU";
  }
  if (millicores === 768) {
    return "3/4 vCPU";
  }
  if (millicores === 1024) {
    return "1 vCPU";
  }

  if (millicores >= 1024 && millicores % 1024 === 0) {
    return `${millicores / 1024} vCPU`;
  }

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
