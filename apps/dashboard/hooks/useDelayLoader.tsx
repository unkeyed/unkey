import { useEffect, useState } from "react";
export const useDelayLoader = (isPending: boolean, delay = 50) => {
  const [showLoader, setShowLoader] = useState(false);

  useEffect(() => {
    let timeout: NodeJS.Timeout;
    if (isPending) {
      timeout = setTimeout(() => {
        setShowLoader(true);
      }, delay);
    } else {
      setShowLoader(false);
    }

    return () => {
      clearTimeout(timeout);
    };
  }, [isPending, delay]);

  return showLoader;
};
