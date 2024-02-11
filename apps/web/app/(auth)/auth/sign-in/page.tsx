import { WorkOS } from "@workos-inc/node";
import Link from "next/link";

const workos = new WorkOS(process.env.WORKOS_API_KEY);
const clientId = process.env.WORKOS_CLIENT_ID!;

export default async function Page() {
  const authorizationUrl = workos.userManagement.getAuthorizationUrl({
    // Specify that we'd like AuthKit to handle the authentication flow
    provider: "authkit",

    // The callback endpoint that WorkOS will redirect to after a user authenticates
    redirectUri: "http://localhost:3000/auth/callback",
    clientId,
  });

  return (
    <>
      <h1>Create an account</h1>

      <Link href={authorizationUrl}>Log in with WorkOs</Link>
    </>
  );
}
