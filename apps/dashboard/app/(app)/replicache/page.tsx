"use client";

import { Button } from "@/components/ui/button";
import { mutators } from "@/lib/replicache/replicache";
import { newId } from "@unkey/id";
import { useState } from "react";
import { TEST_LICENSE_KEY } from "replicache";
import { useReplicache } from "replicache-nextjs/lib/frontend";
import { useSubscribe } from "replicache-react";

export default function Page() {
  const rep = useReplicache({
    name: "replicache",
    pullURL: "/replicache/pull",
    pushURL: "/replicache/push",
    licenseKey: TEST_LICENSE_KEY,

    mutators,
  });

  const apis = useSubscribe(rep, async (tx) => {
    const x = await tx.scan({ prefix: "api/" }).entries().toArray();

    return x;
  });
  if (!rep) {
    return null;
  }
  return (
    <div>
      <h1>Page</h1>

      {apis?.map((api) => (
        <div key={api.id} className="w-full bg-warn flex ">
          {JSON.stringify(api)}{" "}
          <Button
            onClick={() => {
              rep.mutate.deleteApi({ id: api.id });
            }}
          >
            Delete
          </Button>{" "}
        </div>
      ))}

      <Button
        onClick={() => {
          rep.mutate.createApi({
            id: newId("api"),
            name: "hello",
          });
        }}
      >
        Add Api
      </Button>
    </div>
  );
}
