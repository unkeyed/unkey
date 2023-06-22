import { Redis } from "@upstash/redis";
import { redirect } from "next/navigation";

const redis = Redis.fromEnv();

export default async function(props: { params: { id: string } }) {
  const id = props.params.id;
  console.log({ id });
  const email = Buffer.from(id, "base64url").toString();
  console.log({ email });

  await redis
    .pipeline()
    .zadd("early-access", { score: Date.now(), member: email })
    .zrem("waitlist", email)
    .exec();

  return redirect("/app");
}
