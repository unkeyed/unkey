export function getRowClass(severity: string): string {
  const baseClasses = "transition-colors group rounded-md cursor-pointer";

  switch (severity) {
    case "ERROR":
      return `${baseClasses} text-error-11 bg-error-2 hover:bg-error-3`;
    case "WARN":
      return `${baseClasses} text-warning-11 bg-warning-2 hover:bg-warning-3`;
    case "INFO":
      return `${baseClasses} text-info-11 bg-info-2 hover:bg-info-3`;
    case "DEBUG":
    default:
      return `${baseClasses} text-grayA-9 hover:text-accent-11 hover:bg-grayA-3`;
  }
}
