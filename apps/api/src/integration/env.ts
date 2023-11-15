import { z } from "zod"


const requiredEnv = z.object({
  UNKEY_BASE_URL: z.string().url(),
  UNKEY_ROOT_KEY: z.string(),

})

export function testEnv() {
  const res = requiredEnv.safeParse(process.env)
  if (!res.success) {
    throw new Error(`Missing required environment variables: ${res.error.message}`)
  }
  return res.data

}
