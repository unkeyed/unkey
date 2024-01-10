import { auth, signIn, signOut } from "@/auth";
import { GeistMono } from "geist/font/mono";
import { GeistSans } from "geist/font/sans";
import type { Metadata } from "next";
import { Toaster } from "sonner";
import "./globals.css";
import { NavLink } from "./nav";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

export const metadata: Metadata = {
  metadataBase: new URL("https://unkey.dev"),
  title: "AI billing example with Unkey",
  description:
    "Simple AI image generation application. Contains example code of generating and refilling Unkey API keys in response to a Stripe payment link, and using the `remaining` field for measuring usage.",
  openGraph: {
    title: "AI billing example with Unkey",
    images: ["https://unkey.dev/images/templates/unkey-stripe.png"],
  },
};

export default async function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const handleSignIn = async () => {
    "use server";
    await signIn("github");
  };
  const handleSignOut = async () => {
    "use server";
    await signOut();
  };
  const sess = await auth();
  return (
    <html lang="en" className={`${GeistSans.variable} ${GeistMono.variable}`}>
      <body className="bg-gray-100">
        <Toaster />
        <nav className="border-b">
          <div className="flex h-16 items-center px-4">
            <div className="mx-6 flex items-center gap-6">
              <NavLink href="/" label="Generate" />
              <NavLink href="/analytics" label="Analytics" />
              <NavLink href="/credits" label="Credits" />
            </div>
            {/* <div className="ml-auto flex items-center space-x-4 ">
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <button
                    type="button"
                    className="relative h-8 w-8 rounded-full shadow bg-gradient-to-tr from-purple-500 via-orange-300 to-pink-500"
                  >
                    <Avatar className="h-8 w-8">
                      {sess?.user?.image ? (
                        <>
                          {" "}
                          <AvatarImage src={sess.user.image} alt="user" />
                          <AvatarFallback>
                            {sess?.user?.name
                              ?.split(" ")
                              ?.map((s) => s.at(0))
                              ?.join("")}
                          </AvatarFallback>
                        </>
                      ) : null}
                    </Avatar>
                  </button>
                </DropdownMenuTrigger>
                <DropdownMenuContent className="w-56 bg-white" align="end" forceMount>
                  {sess ? (
                    <DropdownMenuLabel className="font-normal">
                      <div className="flex flex-col space-y-1">
                        <p className="text-sm font-medium leading-none">{sess.user?.name}</p>
                        <p className="text-xs leading-none text-muted-foreground">
                          {sess.user?.email}
                        </p>
                      </div>
                    </DropdownMenuLabel>
                  ) : (
                    <form action={handleSignIn}>
                      <button type="submit">Sign In</button>
                    </form>
                  )}

                  {sess ? (
                    <>
                      <DropdownMenuSeparator />
                      <form action={handleSignOut}>
                        <button className="text-xs pl-2.5 pb-2.5" type="submit">
                          Sign Out
                        </button>
                      </form>
                    </>
                  ) : null}
                </DropdownMenuContent>
              </DropdownMenu>
            </div> */}
          </div>
        </nav>
        <main className="container mx-auto mt-8">{children}</main>
      </body>
    </html>
  );
}
