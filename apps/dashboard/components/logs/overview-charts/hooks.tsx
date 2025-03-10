import { useCallback, useEffect, useRef, useState } from "react";
import type { ChartLabels } from "./types";

type WaveAnimationOptions = {
  dataPoints?: number;
  animationSpeed?: number;
  labels: ChartLabels;
};

export function useWaveAnimation({
  dataPoints = 200,
  animationSpeed = 0.02,
  labels,
}: WaveAnimationOptions) {
  const primaryWave = { amplitude: 0.2, baseValue: 0.5 };
  const secondaryWave = { amplitude: 0.12, baseValue: 0.25, frequency: 1.2 };
  const [mockData, setMockData] = useState(() => generateInitialData());
  const [phase, setPhase] = useState(0);
  const animationRef = useRef(0);

  function generateInitialData() {
    return Array.from({ length: dataPoints }).map((_, index) => ({
      [labels.primaryKey]: primaryWave.baseValue,
      [labels.secondaryKey]: secondaryWave.baseValue,
      index,
      originalTimestamp: Date.now(),
    }));
  }

  // Animation frame function with smooth, continuous wave patterns
  // biome-ignore lint/correctness/useExhaustiveDependencies: go touch some grass biome
  const animate = useCallback(() => {
    setPhase((prev) => prev + animationSpeed);
    animationRef.current = requestAnimationFrame(animate);
  }, [animationSpeed]);

  // Start/stop animation with requestAnimationFrame for smoother performance
  useEffect(() => {
    animationRef.current = requestAnimationFrame(animate);

    return () => {
      if (animationRef.current) {
        cancelAnimationFrame(animationRef.current);
      }
    };
  }, [animate]);

  // Update data based on the current phase
  useEffect(() => {
    setMockData((prevData) =>
      prevData.map((item, index) => {
        const baseWavePosition = -phase + index * (Math.PI / 20);

        // Calculate primary wave with fixed defaults
        const primaryWavePosition = baseWavePosition;
        const primaryValue =
          Math.sin(primaryWavePosition) * primaryWave.amplitude + primaryWave.baseValue;

        // Calculate secondary wave with fixed defaults
        const secondaryWavePosition = baseWavePosition * secondaryWave.frequency;
        const secondaryValue =
          Math.sin(secondaryWavePosition) * secondaryWave.amplitude + secondaryWave.baseValue;

        return {
          ...item,
          [labels.primaryKey]: primaryValue,
          [labels.secondaryKey]: secondaryValue,
        };
      }),
    );
  }, [phase, labels.primaryKey, labels.secondaryKey]);

  return {
    mockData,
    currentTime: Date.now(),
  };
}
