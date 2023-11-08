

import { z } from "zod"

export const env = z.object({
  Bindings: z.object({
    DATABASE_HOST: z.string(),
    DATABASE_USERNAME: z.string(),
    DATABASE_PASSWORD: z.string(),
    AXIOM_TOKEN: z.string(),
    RATELIMIT: z.any()
  })

})

export type Env = z.infer<typeof env>
