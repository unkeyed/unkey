import { keys } from "@/server/unkey-client";
import { createKey } from "./actions";

export default function Home() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center p-24 gap-1">
      <h1 className="text-xl font-bold">Weather API</h1>
      <h2>Get access to a realtime 7 day forecast. Keys expire after 1 minute</h2>
      <form action={createKey}>
        <div className="flex flex-col gap-1">
          <button type="submit" className="bg-black px-2 py-1 text-white rounded">
            Create API Key
          </button>
        </div>
      </form>
      <ul className="flex flex-col gap-1 pt-2">
        {keys.map((key) => (
          <li className="border-black border rounded p-2" key={key.keyId}>
            <p className="font-bold">Key: {key.key}</p>
            <p className="text-gray-600 text-sm">Expires on {new Date(key.expires).toString()}</p>
          </li>
        ))}
      </ul>
    </main>
  );
}
