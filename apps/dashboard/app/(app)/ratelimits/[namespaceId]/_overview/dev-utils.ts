export type RatelimitOverviewLogs = {
  identifier: string;
  passed: number;
  blocked: number;
  avgLatency: number;
  p99Latency: number;
  lastRequest: string;
  hasWarning?: boolean;
};

function generateRandomIdentifier() {
  const types = ["User", "IP", "API-Key"];
  const type = types[Math.floor(Math.random() * types.length)];

  switch (type) {
    case "User":
      return `User-${Math.floor(Math.random() * 999)
        .toString()
        .padStart(3, "0")}`;
    case "IP":
      return `IP-${Array.from({ length: 4 }, () => Math.floor(Math.random() * 256)).join(".")}`;
    case "API-Key":
      // biome-ignore lint/correctness/noSwitchDeclarations: <explanation>
      const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ";
      // biome-ignore lint/correctness/noSwitchDeclarations: <explanation>
      const suffix = Array.from(
        { length: 3 },
        () => letters[Math.floor(Math.random() * letters.length)],
      ).join("");
      return `API-Key-${suffix}`;
    default:
      return "unknown";
  }
}

function generateRandomDate(startDate: Date, endDate: Date): string {
  const randomDate = new Date(
    startDate.getTime() + Math.random() * (endDate.getTime() - startDate.getTime()),
  );

  const month = randomDate.toLocaleString("en-US", { month: "short" }).toUpperCase();
  const day = randomDate.getDate().toString().padStart(2, "0");
  const hours = randomDate.getHours().toString().padStart(2, "0");
  const minutes = randomDate.getMinutes().toString().padStart(2, "0");
  const seconds = randomDate.getSeconds().toString().padStart(2, "0");
  const milliseconds = randomDate.getMilliseconds().toString().padStart(2, "0");

  return `${month.slice(0, 3)} ${day} ${hours}:${minutes}:${seconds}.${milliseconds}`;
}

function generateAnomalyData(): Partial<RatelimitOverviewLogs> {
  const anomalyTypes = [
    // High blocked requests
    () => ({
      blocked: Math.floor(Math.random() * 200) + 300,
      hasWarning: true,
    }),
    // Very high latency average
    () => ({ avgLatency: Math.floor(Math.random() * 15) + 10 }),
    // Extremely high P99
    () => ({ p99Latency: Math.floor(Math.random() * 50) + 30 }),
    // High blocked and high latency
    () => ({
      blocked: Math.floor(Math.random() * 150) + 200,
      avgLatency: Math.floor(Math.random() * 10) + 8,
      p99Latency: Math.floor(Math.random() * 20) + 15,
      hasWarning: true,
    }),
    // Very low latency but high P99 (inconsistent performance)
    () => ({
      avgLatency: Math.random() * 0.5,
      p99Latency: Math.floor(Math.random() * 30) + 20,
    }),
  ];

  return anomalyTypes[Math.floor(Math.random() * anomalyTypes.length)]();
}

export function generateMockApiData(numEntries = 50): RatelimitOverviewLogs[] {
  const endDate = new Date();
  const startDate = new Date(endDate.getTime() - 30 * 24 * 60 * 60 * 1000); // 30 days ago

  // Ensure at least 30% of entries have anomalies
  const numAnomalies = Math.max(Math.floor(numEntries * 0.3), 1);
  const anomalyIndices = new Set<number>();
  while (anomalyIndices.size < numAnomalies) {
    anomalyIndices.add(Math.floor(Math.random() * numEntries));
  }

  return Array.from({ length: numEntries }, (_, index) => {
    const baseData = {
      identifier: generateRandomIdentifier(),
      passed: Math.floor(Math.random() * 600000000) + 50,
      blocked: Math.floor(Math.random() * 50000000),
      avgLatency: Number((Math.random() * 3).toFixed(2)),
      p99Latency: Number((Math.random() * 5).toFixed(2)),
      lastRequest: generateRandomDate(startDate, endDate),
      hasWarning: false,
    };

    // Apply anomaly if this index was selected
    if (anomalyIndices.has(index)) {
      const anomaly = generateAnomalyData();
      return { ...baseData, ...anomaly };
    }

    return baseData;
  }).sort((a, b) => {
    const dateA = new Date(
      Date.parse(
        a.lastRequest.replace(/[A-Z]{3}/, (month) => {
          const months = {
            JAN: 0,
            FEB: 1,
            MAR: 2,
            APR: 3,
            MAY: 4,
            JUN: 5,
            JUL: 6,
            AUG: 7,
            SEP: 8,
            OCT: 9,
            NOV: 10,
            DEC: 11,
          };
          return (months[month as keyof typeof months] + 1).toString();
        }),
      ),
    );
    const dateB = new Date(
      Date.parse(
        b.lastRequest.replace(/[A-Z]{3}/, (month) => {
          const months = {
            JAN: 0,
            FEB: 1,
            MAR: 2,
            APR: 3,
            MAY: 4,
            JUN: 5,
            JUL: 6,
            AUG: 7,
            SEP: 8,
            OCT: 9,
            NOV: 10,
            DEC: 11,
          };
          return (months[month as keyof typeof months] + 1).toString();
        }),
      ),
    );
    return dateB.getTime() - dateA.getTime();
  });
}
