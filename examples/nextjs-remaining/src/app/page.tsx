"use client";
import { useEffect, useState } from "react";

import { createKey, getKeys } from "./actions";

type Key = {
  id: string;
  remaining?: number | undefined;
  apiId: string;
};

export default function Home() {
  const [keys, setKeys] = useState<Key[]>([]);
  const [loading, setLoading] = useState<boolean>(false);

  async function fetchKeys() {
    const { keys } = await getKeys();
    setKeys(keys);
  }

  useEffect(() => {
    fetchKeys();
  }, []);

  const onSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setLoading(true);

    const formData = new FormData(e.currentTarget);
    const remaining = formData.get("remaining");
    await createKey(Number(remaining));

    setLoading(false);
    fetchKeys();
  };

  return (
    <main className="flex flex-col justify-center items-center min-h-screen gap-1 p-24">
      <div className="flex flex-col gap-1 items-start w-full max-w-xl">
        <h1 className="text-xl font-bold">Remaining Functionality</h1>
        <h2>Create a new API key and see the remaining functionality of the app.</h2>
        <form onSubmit={onSubmit} className="w-full">
          <div className="flex flex-col gap-3 mt-5">
            <input
              type="number"
              name="remaining"
              className="p-2 border border-black rounded"
              placeholder="Enter the limit amount of requests"
              defaultValue={100}
              min={1}
            />

            <button
              type="submit"
              className="px-2 py-1 text-white bg-black rounded"
              disabled={loading}
            >
              {loading ? "Creating..." : "Create API Key"}
            </button>
          </div>
        </form>
      </div>
      <div className="flex flex-col gap-1 w-full max-w-xl mt-4 ">
        {keys.map((key) => (
          <div className="p-2 border border-black rounded w-full" key={key.id}>
            <p className="font-bold">Key: {key.id}</p>
            <p className="text-sm text-gray-600">Remaining Requests: {key.remaining}</p>
          </div>
        ))}
      </div>
    </main>
  );
}
