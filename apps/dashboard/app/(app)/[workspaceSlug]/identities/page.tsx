import { IdentitiesClient } from "./_components/identities-client";
import { Navigation } from "./navigation";

export default function Page() {
  return (
    <div>
      <Navigation />
      <IdentitiesClient />
    </div>
  );
}
