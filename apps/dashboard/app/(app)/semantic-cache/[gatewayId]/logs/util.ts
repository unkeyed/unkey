export function getInterval(interval: string) {
  const now = new Date();
  console.info({ interval });
  let _timestamp = 0;

  switch (interval) {
    case "24h":
      _timestamp = now.getTime() - 24 * 60 * 60 * 1000; // 24 hours in milliseconds
      break;
    case "7d":
      // Get the start of the day 7 days ago
      _timestamp = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 7).getTime();
      break;
    case "30d":
      // Get the start of the day 30 days ago
      _timestamp = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 30).getTime();
      break;
    case "90d":
      // Get the start of the day 90 days ago
      _timestamp = new Date(now.getFullYear(), now.getMonth(), now.getDate() - 90).getTime();
      break;
    default:
      _timestamp = now.getTime() - 24 * 60 * 60 * 1000; // 24 hours in milliseconds
      break;
  }

  return _timestamp;
}
