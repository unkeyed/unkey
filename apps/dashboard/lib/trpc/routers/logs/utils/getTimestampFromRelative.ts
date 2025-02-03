export const getTimestampFromRelative = (relativeTime: string): number => {
  let totalMilliseconds = 0;

  for (const [, amount, unit] of relativeTime.matchAll(/(\d+)([hdm])/g)) {
    const value = Number.parseInt(amount, 10);

    switch (unit) {
      case "h":
        totalMilliseconds += value * 60 * 60 * 1000;
        break;
      case "d":
        totalMilliseconds += value * 24 * 60 * 60 * 1000;
        break;
      case "m":
        totalMilliseconds += value * 60 * 1000;
        break;
    }
  }

  return Date.now() - totalMilliseconds;
};
