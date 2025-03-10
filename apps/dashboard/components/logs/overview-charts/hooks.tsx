import { useCallback, useEffect, useRef, useState } from "react";
import type { ChartLabels } from "./types";

type WaveAnimationOptions = {
  dataPoints?: number;
  animationSpeed?: number;
  labels: ChartLabels;
  animate?: boolean; // New option to control animation
};

export function useWaveAnimation({
  dataPoints = 200,
  animationSpeed = 0.02,
  labels,
  animate = true, // Default to true for backward compatibility
}: WaveAnimationOptions) {
  const primaryWave = { amplitude: 0.2, baseValue: 0.5 };
  const secondaryWave = { amplitude: 0.12, baseValue: 0.25, frequency: 1.2 };
  const [mockData, setMockData] = useState(() => generateInitialData());
  const [phase, setPhase] = useState(0);
  const animationRef = useRef(0);

  function generateInitialData() {
    return Array.from({ length: dataPoints }).map((_, index) => {
      // If not animating, calculate the initial wave values instead of base values
      if (!animate) {
        const baseWavePosition = index * (Math.PI / 20);

        // Calculate primary wave
        const primaryWavePosition = baseWavePosition;
        const primaryValue =
          Math.sin(primaryWavePosition) * primaryWave.amplitude + primaryWave.baseValue;

        // Calculate secondary wave
        const secondaryWavePosition = baseWavePosition * secondaryWave.frequency;
        const secondaryValue =
          Math.sin(secondaryWavePosition) * secondaryWave.amplitude + secondaryWave.baseValue;

        return {
          [labels.primaryKey]: primaryValue,
          [labels.secondaryKey]: secondaryValue,
          index,
          originalTimestamp: Date.now(),
        };
      }

      // Default behavior for animation mode
      return {
        [labels.primaryKey]: primaryWave.baseValue,
        [labels.secondaryKey]: secondaryWave.baseValue,
        index,
        originalTimestamp: Date.now(),
      };
    });
  }

  // Animation frame function with smooth, continuous wave patterns
  // biome-ignore lint/correctness/useExhaustiveDependencies: go touch some grass biome
  const animateFrame = useCallback(() => {
    setPhase((prev) => prev + animationSpeed);
    animationRef.current = requestAnimationFrame(animateFrame);
  }, [animationSpeed]);

  // Start/stop animation with requestAnimationFrame for smoother performance
  useEffect(() => {
    // Only start animation if animate flag is true
    if (animate) {
      animationRef.current = requestAnimationFrame(animateFrame);
      return () => {
        if (animationRef.current) {
          cancelAnimationFrame(animationRef.current);
        }
      };
    }
    // If not animating, just return cleanup function
    return () => {
      if (animationRef.current) {
        cancelAnimationFrame(animationRef.current);
      }
    };
  }, [animate, animateFrame]);

  // Update data based on the current phase
  useEffect(() => {
    // Only update data if we're animating
    if (!animate) {
      return;
    }

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
  }, [phase, labels.primaryKey, labels.secondaryKey, animate]);

  return {
    mockData,
    currentTime: Date.now(),
  };
}
