"use client";

import { Button } from "@/components/ui/button";
import { useApis, useReplicache } from "@/lib/replicache/hooks";
import { newId } from "@unkey/id";

export default function Page() {
  const r = useReplicache();
  const apis = useApis();

  return (
    <div>
      <h1>Page</h1>

      {Object.entries(apis).map(([key, api]) => (
        <div key={key} className="w-full bg-warn flex ">
          {JSON.stringify(api)}{" "}
          <Button
            onClick={() => {
              r!.mutate.deleteApi({ id: api.id });
            }}
          >
            Delete
          </Button>{" "}
        </div>
      ))}

      <Button
        onClick={() => {
          r?.mutate.createApi({
            id: newId("api"),
            name: "api name",
          });
        }}
      >
        Add Api
      </Button>
    </div>
  );
}
