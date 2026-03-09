export type FormattedParts = { value: string; unit: string };

export function formatCpuParts(millicores: number): FormattedParts {
  if (millicores === 0) {
    return { value: "—", unit: "" };
  }
  if (millicores === 256) {
    return { value: "1/4", unit: "vCPU" };
  }
  if (millicores === 512) {
    return { value: "1/2", unit: "vCPU" };
  }
  if (millicores === 768) {
    return { value: "3/4", unit: "vCPU" };
  }
  if (millicores === 1024) {
    return { value: "1", unit: "vCPU" };
  }
  if (millicores >= 1024 && millicores % 1024 === 0) {
    return { value: `${millicores / 1024}`, unit: "vCPU" };
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
