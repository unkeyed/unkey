import Chat from "@/components/chat/chat";

export default function Home() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center  md:p-24">
      <h1 className="mb-4 text-2xl md:text-3xl">
        UnkeyGPT <span className="motion-safe:animate-pulse">📣</span>
      </h1>
      <Chat />
    </main>
  );
}
