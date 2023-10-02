
import { UnkeyElements } from "./keys/client"

export default function Home() {
  
  return (
    <main className="flex min-h-screen flex-col items-center justify-between p-24">
      <div className="flex flex-col items-center justify-center">
        <h1 className="text-4xl font-bold">Welcome to the Unkey + Auth Provider</h1>
        <p className="text-xl mt-4">This is a demo of how you can use Unkey to secure your API with an Auth Provider.</p>
        <UnkeyElements />
      </div>

        

    </main>
  )
}
