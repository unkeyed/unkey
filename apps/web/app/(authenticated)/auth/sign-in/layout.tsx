import { auth } from "@clerk/nextjs"
import { redirect } from "next/navigation";

export default function AuthLayout(props: { children: React.ReactNode }) {
  const { userId } = auth();

  if (userId) {
    redirect("/app")
  }
  return (
    <>
      <div className="grid h-screen place-items-center">
        <div className="w-full">{props.children}</div>
      </div>
    </>
  );
}
