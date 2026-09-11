export function AccountUnavailable({ reason }: { reason: "local" | "impersonation" }) {
  return (
    <section
      className="rounded-lg border border-grayA-4 bg-grayA-2 p-6"
      aria-labelledby="account-unavailable"
    >
      <h2 id="account-unavailable" className="font-medium">
        Account settings unavailable
      </h2>
      <p className="mt-2 text-sm text-gray-11">
        {reason === "local"
          ? "Managed profile and security settings are available when the dashboard uses WorkOS authentication."
          : "Account changes are disabled while you are impersonating a user. Stop impersonating to manage this account."}
      </p>
    </section>
  );
}
