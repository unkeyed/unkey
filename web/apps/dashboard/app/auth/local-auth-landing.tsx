import Link from "next/link";

export function LocalAuthLanding({ description }: { description: string }) {
  return (
    <main className="flex min-h-screen items-center justify-center px-6">
      <section className="max-w-md space-y-4 text-center">
        <h1 className="font-semibold text-2xl">Local dashboard</h1>
        <p className="text-accent-9 text-sm">{description}</p>
        <Link
          className="inline-flex h-9 items-center justify-center rounded-md bg-white px-4 font-medium text-black text-sm hover:bg-white/90"
          href="/apis"
        >
          Continue to dashboard
        </Link>
      </section>
    </main>
  );
}
