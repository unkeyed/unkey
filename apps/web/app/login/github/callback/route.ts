// app/login/github/callback/route.ts
import { github, lucia } from "@/lib/auth/index";
import { db, schema } from "@/lib/db";
import { newId } from "@unkey/id";
import { OAuth2RequestError } from "arctic";
import { generateId } from "lucia";
import { cookies } from "next/headers";

export async function GET(request: Request): Promise<Response> {
  const url = new URL(request.url);
  const code = url.searchParams.get("code");
  const state = url.searchParams.get("state");
  const storedState = cookies().get("github_oauth_state")?.value ?? null;
  if (!code || !state || !storedState || state !== storedState) {
    return new Response(null, {
      status: 400,
    });
  }

  try {
    const tokens = await github.validateAuthorizationCode(code);
    const githubUserResponse = await fetch("https://api.github.com/user", {
      headers: {
        Authorization: `Bearer ${tokens.accessToken}`,
      },
    });
    const githubUser: GitHubUser = await githubUserResponse.json();

    // Replace this with your own DB client.
    const oauth = await db.query.oauth.findFirst({
      where: (table, { eq, and }) => and(eq(table.provider, "github"), eq(table.id, githubUser.id)),
      with: { user: true },
    });

    if (oauth) {
      const session = await lucia.createSession(oauth.user.id, {});
      const sessionCookie = lucia.createSessionCookie(session.id);
      cookies().set(sessionCookie.name, sessionCookie.value, sessionCookie.attributes);
      return new Response(null, {
        status: 302,
        headers: {
          Location: "/",
        },
      });
    }

    const userId = newId("user");

    // Replace this with your own DB client.
    await db.insert(schema.users).values({
      id: userId,
      createdAt: new Date(),
      email: githubUser.login,
      firstName: githubUser.login,
      lastName: githubUser.login,
      profilePictureUrl: githubUser.login,
      updatedAt: new Date(),
    });

    await db.insert(schema.oauth).values({
      id: githubUser.id,
      createdAt: new Date(),
      provider: "github",
      userId,
    });

    const session = await lucia.createSession(userId, {});
    const sessionCookie = lucia.createSessionCookie(session.id);
    cookies().set(sessionCookie.name, sessionCookie.value, sessionCookie.attributes);
    return new Response(null, {
      status: 302,
      headers: {
        Location: "/",
      },
    });
  } catch (e) {
    // the specific error message depends on the provider
    if (e instanceof OAuth2RequestError) {
      // invalid code
      return new Response(null, {
        status: 400,
      });
    }
    return new Response(null, {
      status: 500,
    });
  }
}

interface GitHubUser {
  id: string;
  login: string;
}
