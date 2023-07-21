export function formatDate(dateString: string) {
  let parts = dateString.split("-");
  let hasDay = parts.length > 2;

  return new Date(`${dateString}Z`).toLocaleDateString("en-US", {
    day: hasDay ? "numeric" : undefined,
    month: "long",
    year: "numeric",
    timeZone: "UTC",
  });
}
