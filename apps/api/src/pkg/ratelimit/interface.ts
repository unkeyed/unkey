
import { Result } from "@unkey/result"

export interface Ratelimiter {
  limit: (req: RatelimitRequest) => Promise<Result<RatelimitResponse>>
}



export type RatelimitRequest = {
  keyId: string
  windowSizeMs: number
  limit: number
}
export type RatelimitResponse = {
  currentLimit: number
  reset: number
  pass: boolean
}
