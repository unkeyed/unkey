const messages = {
  monday: [
    "The week stretches before you like an endless void.",
    "4 days. You can do this. Probably.",
    "Monday is just Friday's origin story.",
    "Congratulations, you survived the weekend. Now survive this.",
    "Monday called. It wants your soul.",
    "Plot twist: It's still not Friday.",
    "You're 0% through the week. Enjoy!",
    "Monday: Because even calendars need a villain.",
    "The longest journey begins with a single Monday.",
    "Coffee won't fix this, but it's a start.",
    "Welcome back to your regularly scheduled suffering.",
    "Monday is proof that time is a social construct.",
    "Fun fact: Monday is the farthest point from the next weekend.",
    "At least it's not Monday morning anymore. Oh wait.",
    "Monday tip: Lower your expectations. Lower. Keep going.",
    "This Monday is sponsored by existential dread.",
    "Remember when it was the weekend? Good times.",
    "Monday: 1, You: 0. But there's still time.",
    "If Monday had a face, we'd all agree.",
    "You've got this! (This is what I'm programmed to say.)",
  ],
  tuesday: [
    "It's not Monday anymore. Small wins.",
    "Tuesday: Monday's annoying sequel.",
    "Still not Friday. Shocking, I know.",
    "Tuesday is just Monday cosplaying as progress.",
    "You're like 20% done. Basically nothing. Sorry.",
    "The week is a tunnel. You cannot see the light yet.",
    "Tuesday: When Monday's hangover finally clears.",
    "At least you can say 'this week' and mean something now.",
    "Tuesday is Monday's less evil twin.",
    "Only 3 more days of pretending to be productive.",
    "Fun fact: Nobody has ever been excited about Tuesday.",
    "Tuesday: The forgotten middle child of the week.",
    "Still closer to last Friday than next Friday. Oof.",
    "You've made it 24 hours. Only 72 more to go!",
    "Tuesday called. It has nothing interesting to say.",
    "If the week were a movie, this would be the slow part.",
    "Tuesday is just a Monday that went to therapy.",
    "Halfway to Wednesday. That's something, right?",
    "The week is a pie and you've eaten one slice.",
    "Keep going. Or don't. Tuesday doesn't care.",
  ],
  wednesday: [
    "Halfway there. Woooah, livin' on a prayer.",
    "The hump day is real and you're climbing it.",
    "Wednesday: Close enough to see Friday, too far to taste it.",
    "You're at the peak. It's all downhill from here!",
    "50% complete. Would not recommend so far.",
    "Wednesday is Friday's trailer.",
    "The week has a middle and you're standing in it.",
    "Hump day: The only day that sounds inappropriate and isn't.",
    "You can officially say 'later this week' about Friday.",
    "Wednesday: When hope begins to flicker.",
    "You're closer to Friday than Monday. Let that sink in.",
    "Over the hump. Coasting is now acceptable.",
    "Wednesday is like the middle seat of days.",
    "If the week were a sandwich, this would be the meat.",
    "Midweek crisis? Totally normal.",
    "Two days down, two to go. Math checks out.",
    "Wednesday: Because someone had to be in the middle.",
    "You've reached the week's equator.",
    "The downhill slide begins now.",
    "We're gonna make it. Probably.",
  ],
  thursday: [
    "Tomorrow is Friday. You can almost smell the weekend.",
    "Thursday is Friday Eve. Prepare accordingly.",
    "One more sleep until Friday!",
    "So close you can taste it. Tastes like freedom.",
    "Thursday: The light at the end of the tunnel is visible.",
    "Pre-Friday vibes activated.",
    "You're 80% there. Home stretch!",
    "Thursday is basically Friday's waiting room.",
    "The weekend is loading... 80% complete.",
    "Tomorrow is the promised land.",
    "Thursday: When Friday starts calling your name.",
    "One more day. You're basically already there.",
    "Thursday energy: Checked out mentally, present physically.",
    "Can you hear it? That's Friday whispering.",
    "The end is nigh (in a good way).",
    "Thursday: Too late to start anything, too early to celebrate.",
    "You've almost survived another week. Congrats!",
    "Friday is so close, I can see its Instagram stories.",
    "One more sunrise until freedom.",
    "Thursday: The final boss before the weekend.",
  ],
  friday: [
    "IT'S FRIDAY! 🎉",
    "Friday has arrived. You made it, you beautiful human.",
    "The weekend awaits. Go forth and touch grass.",
    "FRIDAY! The king of days has arrived!",
    "You did it! The week is defeated!",
    "Friday: When productivity becomes optional.",
    "The prophecy is fulfilled. It is Friday.",
    "Pop the champagne (or soda, no judgment).",
    "Friday: The day emails can wait until Monday.",
    "Weekend mode: ENGAGED.",
    "It's Friday! Everything is possible!",
    "The week bows to you. You've conquered it.",
    "Friday: The finish line of adult life.",
    "Time to make poor decisions you'll recover from by Monday.",
    "Friday vibes only. No exceptions.",
    "You survived! Achievement unlocked: Weekend.",
    "It's finally Friday, and you're finally free.",
    "Friday: Because you've earned it.",
    "The best day of the week has arrived!",
    "TGIF is not just letters. It's a lifestyle.",
  ],
  weekend: [
    "It's the weekend! Why are you hitting this API?",
    "Go outside. The API will be here Monday.",
    "Weekend mode: activated. Chill mode: engaged.",
    "It's the weekend! Close the laptop!",
    "You're working on a weekend? We need to talk.",
    "Weekend detected. Please step away from the keyboard.",
    "REST is for APIs and for you, right now.",
    "The weekend is not for checking APIs. Go brunch.",
    "Saturday/Sunday detected. Touch grass immediately.",
    "Why are you here? The weekend is out there!",
    "This API respects work-life balance. Do you?",
    "Weekend vibes: No thoughts, only vibes.",
    "It's the weekend. Your code can wait.",
    "404: Productivity not found (as it should be).",
    "The weekend fairy says log off.",
    "Weekends are for recovering, not for APIs.",
    "Plot twist: It's the weekend. Go enjoy it.",
    "This is a judgment-free zone, but also: it's the weekend.",
    "Your weekend is too precious for HTTP requests.",
    "Come back Monday. Or don't. Live your life.",
  ],
};

function getDayKey(day: number): keyof typeof messages {
  if (day === 0 || day === 6) return "weekend";
  return (["", "monday", "tuesday", "wednesday", "thursday", "friday"] as const)[day];
}

function pickRandom<T>(arr: T[]): T {
  return arr[Math.floor(Math.random() * arr.length)];
}

const server = Bun.serve({
  port: process.env.PORT || 3000,
  fetch(req) {
    const url = new URL(req.url);

    if (url.pathname !== "/") {
      return new Response("Not found", { status: 404 });
    }

    const tz = url.searchParams.get("tz") || "UTC";
    let now: Date;

    try {
      now = new Date(new Date().toLocaleString("en-US", { timeZone: tz }));
    } catch {
      now = new Date();
    }

    const day = now.getDay();
    const dayKey = getDayKey(day);

    const response = {
      isFriday: day === 5,
      message: pickRandom(messages[dayKey]),
    };

    return new Response(JSON.stringify(response), {
      headers: { "Content-Type": "application/json" },
    });
  },
});

console.log(`🗓️  Is It Friday Yet? running on port ${server.port}`);
