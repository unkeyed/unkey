export const formatDuration = (ms: number) => {
  const seconds = ms / 1000;

  if (seconds < 60) {
    return { value: seconds, unit: "second" };
  }

  const minutes = seconds / 60;
  if (minutes < 60) {
    return { value: minutes, unit: "minute" };
  }

  const hours = minutes / 60;
  if (hours < 24) {
    return { value: hours, unit: "hour" };
  }

  const days = hours / 24;
  return { value: days, unit: "day" };
};
