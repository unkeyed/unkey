import { keys } from "@/server/keys";
import { createKey } from "./actions";

export default function Home() {
  return (
    <main className="flex flex-col items-center justify-center min-h-screen gap-1 p-24">
      <h1 className="text-xl font-bold">Weather API</h1>
      <h2>Get access to a realtime 7 day forecast. Keys expire after 1 minute</h2>
      <form action={createKey}>
        <div className="flex flex-col gap-1">
          <button type="submit" className="px-2 py-1 text-white bg-black rounded">
            Create API Key
          </button>
        </div>
      </form>
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
