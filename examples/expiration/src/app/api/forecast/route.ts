import { verifyKey } from "@unkey/api";

type Forcast = {
  date: string;
  tempHigh: number;
  tempLow: number;
  chanceOfPrecipitation: number;
  precipitationKind?: "rain" | "snow";
  humidity: number;
  cloudiness: number;
  summary: string;
};

function getRandomNumber(min: number, max: number) {
  return Math.floor(Math.random() * (max - min) + min);
}

function generateForecast() {
  const sevenDayForcast: Forcast[] = [];

  const today = new Date();

  for (let i = 0; i < 7; i++) {
    const tempHigh = getRandomNumber(-20, 100);
    const tempLow = tempHigh - getRandomNumber(0, 20);

    const cloudiness = getRandomNumber(0, 100);

    const chanceOfPrecipitation = cloudiness > 50 ? getRandomNumber(0, 100) : 0;

    const humidity = getRandomNumber(0, 100);

    let precipitationKind: Forcast["precipitationKind"];

    if (chanceOfPrecipitation > 50 && tempLow < 32) {
      precipitationKind = "snow";
    } else if (chanceOfPrecipitation > 50 && tempLow > 32) {
      precipitationKind = "rain";
    }

    const cloudinessDescriptor =
      cloudiness < 10 ? "sunny" : cloudiness < 50 ? "partly cloudy" : "cloudy";

    const summary = `It's a ${cloudinessDescriptor} day today with a high of ${tempHigh} and a low of ${tempLow}. There is a ${chanceOfPrecipitation}% chance of ${
      typeof precipitationKind === "string"
        ? precipitationKind
        : "precipitation"
    } and a humidity of ${humidity}%.`;

    const date = new Date(today);
    date.setDate(today.getDate() + i);

    sevenDayForcast.push({
      date: date.toISOString(),
      tempHigh,
      tempLow,
      chanceOfPrecipitation,
      precipitationKind,
      humidity,
      cloudiness,
      summary,
    });
  }

  return sevenDayForcast;
}

export async function GET() {
  const forecast = generateForecast();

  return Response.json(forecast, { status: 200 });
}
