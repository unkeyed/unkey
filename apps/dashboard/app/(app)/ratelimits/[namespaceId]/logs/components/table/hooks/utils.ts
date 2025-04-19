import { useEffect, useState } from "react";

let QUERY_TIMESTAMP = Date.now();
let subscribers: ((timestamp: number) => void)[] = [];

export const getQueryTimestamp = () => QUERY_TIMESTAMP;

export const refreshQueryTimestamp = () => {
  QUERY_TIMESTAMP = Date.now();
  subscribers.forEach((callback) => callback(QUERY_TIMESTAMP));
  return QUERY_TIMESTAMP;
};

export const useQueryTimestamp = () => {
  const [timestamp, setTimestamp] = useState(QUERY_TIMESTAMP);

  useEffect(() => {
    const callback = (newTimestamp: number) => setTimestamp(newTimestamp);
    subscribers.push(callback);

    return () => {
      subscribers = subscribers.filter((cb) => cb !== callback);
    };
  }, []);

  return timestamp;
};
