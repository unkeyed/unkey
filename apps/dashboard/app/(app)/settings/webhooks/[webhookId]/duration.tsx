"use client";
import ms from "ms";
import { useEffect, useState } from "react";
type Props = {
  time: number;
};

export const Until: React.FC<Props> = ({ time }) => {
  const [duration, setDuration] = useState<number>(time - Date.now());

  useEffect(() => {
    const interval = setInterval(() => {
      setDuration(time - Date.now());
    }, 1000);

    return () => clearInterval(interval);
  }, [time]);

  if (duration < 0) {
    return <span>now</span>;
  }

  return <span>{ms(duration)}</span>;
};
