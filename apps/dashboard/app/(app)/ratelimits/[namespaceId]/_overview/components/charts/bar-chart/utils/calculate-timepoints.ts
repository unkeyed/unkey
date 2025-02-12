export const calculateTimePoints = (startTime: number, endTime: number) => {
  const points = 5;
  const timeRange = endTime - startTime;
  const step = Math.floor(timeRange / (points - 1));

  return Array.from({ length: points }, (_, i) => new Date(startTime + step * i)).filter(
    (date) => date.getTime() <= endTime,
  );
};
