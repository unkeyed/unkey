import type { Unkey } from '@unkey/api'

declare module 'h3' {
  interface H3EventContext {
    unkey?: Awaited<ReturnType<Unkey['keys']['verify']>>
  }
}
