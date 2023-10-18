import { FadeIn } from "@/components/landing/fade-in";
import { auth } from "@clerk/nextjs";
import { redirect } from "next/navigation";

export default function AuthLayout(props: { children: React.ReactNode }) {
  const { userId } = auth();

  if (userId) {
    return redirect("/app/apis");
  }
  return (
    <FadeIn>
      <div className="grid grid-cols-1  h-screen place-items-center bg-white">
        <div className="container">{props.children}</div>
      </div>
    </FadeIn>
  );
}
