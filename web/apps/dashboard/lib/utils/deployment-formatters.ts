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
