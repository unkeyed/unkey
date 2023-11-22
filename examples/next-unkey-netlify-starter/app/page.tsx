import JokeGenerator from "./joke-generator";
import SignOut from "./sign-out";

export default function Home() {
  return (
    <main className="flex relative min-h-screen flex-col items-center justify-center p-24">
      <div className="absolute top-10 right-10">
        <SignOut />
      </div>
      <JokeGenerator />
    </main>
  );
}
