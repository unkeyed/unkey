import { helloWorld } from "@/lib/flags";

export default async function FlagsDemoPage() {
  const enabled = await helloWorld();
  return (
    <div className="p-6">
      <h1 className="text-lg font-medium">Flags demo</h1>
      <p className="mt-2">hello-world: {enabled ? "on" : "off"}</p>
    </div>
  );
}
