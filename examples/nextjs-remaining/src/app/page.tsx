"use client";
import { useState } from "react";

import { keys } from "@/server/keys";
import { createKey } from "./actions";

export default function Home() {
  const [noOfRequests, setNoOfRequests] = useState(100);

  return (
    <main className="flex flex-col justify-center items-center min-h-screen gap-1 p-24">
      <div className="flex flex-col gap-1 max-w-xl">
        <h1 className="text-xl font-bold">Remaining Functionality</h1>
        <h2>Create a new API key and see the remaining functionality of the app.</h2>
        <form action={createKey}>
          <div className="flex flex-col gap-5 mt-5">
            <input
              type="number"
              name="key"
              className="p-2 border border-black rounded"
              placeholder="Enter the limit amount of requests"
              value={noOfRequests}
              onChange={(e) => setNoOfRequests(Number(e.target.value))}
              min={1}
            />

            <button type="submit" className="px-2 py-1 text-white bg-black rounded">
              Create API Key
            </button>
          </div>
        </form>
      </div>
      <ul className="flex flex-col gap-1 pt-2">
        {keys.map((key) => (
          <li className="p-2 border border-black rounded" key={key.keyId}>
            <p className="font-bold">Key: {key.key}</p>
            <p className="text-sm text-gray-600">Expires on {new Date(key.expires).toString()}</p>
          </li>
        ))}
      </ul>
    </main>
  );
}
