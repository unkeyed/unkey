import { Unkey } from '@unkey/api'
import type { H3Event } from 'h3'
import { useRuntimeConfig } from '#imports'

let unkey: Unkey

export const useUnkey = (event?: H3Event) => {
  if (unkey) return unkey

  const config = useRuntimeConfig(event)
  unkey = new Unkey({ token: config.unkey.key })

  return unkey
}
