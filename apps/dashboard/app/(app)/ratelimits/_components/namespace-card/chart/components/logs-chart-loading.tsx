import { useEffect, useState } from "react";
import { Bar, BarChart, ResponsiveContainer, YAxis } from "recharts";

export const LogsChartLoading = () => {
  const [mockData, setMockData] = useState(generateInitialData());

  function generateInitialData() {
    return Array.from({ length: 100 }).map(() => ({
      success: Math.random() * 0.5 + 0.5,
      error: Math.random() * 0.3,
      originalTimestamp: Date.now(),
    }));
  }

  useEffect(() => {
    const interval = setInterval(() => {
      setMockData((prevData) =>
        prevData.map((item) => ({
          ...item,
          success: Math.random() * 0.5 + 0.5,
          error: Math.random() * 0.3,
        })),
      );
    }, 600); // Update every 200ms for smooth animation

    return () => clearInterval(interval);
  }, []);

  return (
    <div className="flex flex-col h-full animate-pulse">
      <div className="flex-1 min-h-0">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={mockData} margin={{ top: 0, right: 0, bottom: 0, left: 0 }}>
            <YAxis domain={[0, (dataMax: number) => dataMax * 2]} hide />
            <Bar
              dataKey="success"
              fill="hsl(var(--accent-3))"
              stackId="a"
              animationDuration={600}
              isAnimationActive
            />
            <Bar
              dataKey="error"
              fill="hsl(var(--accent-3))"
              stackId="a"
              animationDuration={600}
              isAnimationActive
            />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
};
