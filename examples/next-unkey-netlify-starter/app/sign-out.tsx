import { Button } from "@/components/ui/button";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";

const SignOut = () => {
  const signOutAction = async () => {
    "use server";
    cookies().delete("unkey-token");
    redirect("/auth");
  };
  return (
    <form action={signOutAction}>
      <Button type="submit"> Log Out</Button>
    </form>
  );
};

export default SignOut;
