"use client";
import { type PropsWithChildren, createContext, useContext, useEffect, useState } from "react";
import { Replicache, TEST_LICENSE_KEY } from "replicache";
import { type Mutators, mutators } from "./mutations";

export const ReplicacheContext = createContext<Replicache<Mutators> | null>(null);

export const ReplicacheProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [replicache, setReplicache] = useState<Replicache<Mutators> | null>(null);

  useEffect(() => {
    const r = new Replicache({
      name: "replicache",
      pullURL: "/replicache/pull",
      pushURL: "/replicache/push",
      licenseKey: process.env.NEXT_PUBLIC_REPLICACHE_LICENSE_KEY ?? TEST_LICENSE_KEY,

      mutators,
    });
    setReplicache(r);

    return () => {
      r.close();
    };
  }, []);

  if (replicache === null) {
    return <div className="w-screen h-screen flex items-center justify-center">Loading...</div>;
  }

  return <ReplicacheContext.Provider value={replicache}>{children}</ReplicacheContext.Provider>;
};
