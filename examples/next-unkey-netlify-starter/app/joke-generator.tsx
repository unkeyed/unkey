"use client";
import { Button } from "@/components/ui/button";
import { useState } from "react";
const terribleDadJokes = [
  "Why don't skeletons fight each other? Because they don't have the guts.",
  "I told my wife she should embrace her mistakes. She gave me a hug.",
  "Did you hear about the guy who invented Lifesavers? He made a mint.",
  "What did the janitor say when he jumped out of the closet? Supplies!",
  "I only know 25 letters of the alphabet. I don't know y.",
  "Why don't scientists trust atoms? Because they make up everything.",
  "What did one hat say to the other? Stay here, I'm going on ahead.",
  "Why don't oysters donate to charity? Because they are shellfish.",
  "How does a penguin build its house? Igloos it together.",
  "What do you call fake spaghetti? An impasta.",
  "I used to play piano by ear, but now I use my hands.",
  "Why couldn't the bicycle stand up by itself? Because it was two-tired.",
  "I told my computer I needed a break, and now it won't stop sending me vacation ads.",
  "Did you hear about the mathematician who’s afraid of negative numbers? He'll stop at nothing to avoid them.",
  "Why don't scientists trust atoms? Because they make up everything.",
  "What do you call a fake noodle? An impasta.",
  "Why don't scientists trust atoms? Because they make up everything.",
  "Why don't seagulls fly over the bay? Because then they'd be bagels.",
  "I only know 25 letters of the alphabet. I don't know y.",
  "What do you call a fish wearing a crown? A kingfish.",
];
const JokeGenerator = () => {
  const [joke, setJoke] = useState("");
  const jokeGenerator = async () => {
    const nj = terribleDadJokes[Math.floor(Math.random() * terribleDadJokes.length)];
    setJoke(nj);
  };
  return (
    <div className="w-1/2">
      <Button className="my-4" onClick={jokeGenerator}>
        Generate bad dad joke
      </Button>
      <p className="text-lg">{joke.length > 0 ? joke : "🤔"}</p>
    </div>
  );
};

export default JokeGenerator;
