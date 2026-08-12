import { LocalAuthLanding } from "../../local-auth-landing";

export function LocalSignInPage() {
  return (
    <LocalAuthLanding description="Authentication is disabled in local mode. Continue with the built-in local account." />
  );
}
