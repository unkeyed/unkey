import { eq } from "drizzle-orm";
import { initDB } from "./src/db";
import { schema } from "@unkey/db";
async function main() {
  // const db = await initDB();

  // const keys = await db.query.keys.findMany()
  // console.log({ keys })
  // const workspaces = new Map<string, boolean>()
  // for (const key of keys) {
  //     workspaces.set(key.workspaceId, true)
  // }

  // const tenantIds: string[] = []

  // for (const workspaceId of workspaces.keys()) {
  //     const ws = await db.query.workspaces.findFirst({ where: eq(schema.workspaces.id, workspaceId) })
  //     if (ws) {
  //         console.log(ws)
  //         tenantIds.push(ws.tenantId!)
  //     }
  // }

  // console.log(JSON.stringify(tenantIds,null,2))

  const userIds = [
    "user_2RDKnCMXpG745GJgQgsBt4qN7rF",
    "user_2Rk4FiyXVEqadcjwuIJ7Cly0nYM",
    "user_2Rf7B7uBItVrT9ektUtoXYsEck7",
    "user_2RVbqrZyosFXegyd1betctao0jn",
    "user_2RiY1FETztzaXOa3BZdSjaN9Sf3",
    "user_2RYWqNQUYX6fsDDd7r1X7SZw0vk",
    "user_2RJ4sfNgOcpFYUUI0GSeHZhJghP",
    "user_2RYW3Jk5QlFzb9jUtXPtgX6cl6d",
    "user_2RTzSHOkbqotMLieFvXwgL96N4y",
    "user_2RU3NzWelsBTTwZYqTaTvBZnMW6",
    "user_2RfxggqoeRWjuCMUafhns6xdwhM",
    "user_2RileuGrTOauFSkc8vJ3ic6n07o",
    "user_2RVIXX0IUymIoLviSlTFwM05FpR",
    "user_2RU1G2UqjnzccZIO0dGnhz2Dyih",
    "user_2RUnqndDxZytNhejupDZN2M9YY4",
    "user_2RVku5wALvA8zwp4q7jIZqLoYBT",
    "user_2RfVdZzkOOfY5SupWQMAh5nduHo",
    "user_2RWRZA6uAyzdYbvluacNhx5pTDn",
    "user_2RWFcoxcUNTsaYUcfbIFJM4q1uA",
    "user_2Rb8s4ZVPvVpLJmhf5qb4stnxCL",
    "user_2Rae7ns6B43MFDNEJRoIaqfRsXC",
    "user_2RWR6tUmQC24MLSKLMqMkMxYnuP",
    "user_2RU0yB4qaKf4bU6z87mvuQ3Sgq6",
    "user_2RUAKlHD5PebHueFGgCMYZZfYsD",
    "user_2RYsS6ylV8h9LnBEPgrefgqoY8B",
    "user_2RUPU6AlLKLfVqcmX2NKY8GXMhP",
    "user_2RUoe1e1cL77gx1Mx6CxkMlu7nX",
    "user_2RUJrlz8YSXbGCWqA0yd6YfcvnB",
    "user_2RWqSpz1gqx60qrPTxF4fBBDlBd",
    "user_2RWU8cTrGotWSy7pmKvhhyfLuAI",
    "user_2RVDTGPTChIdCVVm5QUVMtuRk9h",
    "user_2RWaT6b5Qk0u1BDeGIN9lsBHhHY",
    "user_2RU0sAj9jACbq4JuuDXd8GhQYS3",
    "user_2RlH3dOF7D9FBSniTAqduzUSvVc",
    "user_2RkLKxOq14mfWboqMf8vSQvmJrR",
    "user_2RZDNSfB1JefLSmJxyVPNIJGgNn",
    "user_2RUHOjDpDR5mWppfD01tnb7kxWo",
    "user_2RaZsaDb7rFWuRA8Mvt9Usr9MVV",
    "user_2RVpNZ6odEL8I1QWsHcOlWVrriX",
    "user_2RWqh6mi2gXEBA3OUeM0iQcq8CR",
    "user_2RTyJ0k1uL2IC9pbvCmr1RFpQuJ",
    "user_2RY059OGJeeD6jT0OdNX5P1Pgsg",
    "user_2RlSONgprz97x5soneQUiyFyU5X",
    "user_2RXcy72gE5MBHNAESETZ5U09qhZ",
    "user_2RWhfnV4QlRyJVlOiLV0gXP6NV6",
    "user_2RUOcGBwpxDYv2LyloanizLtwgl",
    "user_2RUtQrNOWwiw96mAD2X8qhHQWUL",
    "user_2RYoKUJEbkcPJHo1RMA9XecmWSi",
    "user_2RZF345jY1i05xe3db85vGGbPIX",
    "user_2RWPkdUAICt9jf9HKvcQwSUoWdI",
    "user_2RVxNWMZVfSFhpk3zCgxZjmKqz2",
    "user_2RX6z7WNYSK3ScyMA3LxtBEmeQU",
    "user_2RU2FUB7IkJwNtdGHAiwtZY2fJZ",
    "user_2RVdqFvMgIbxSiiLUrw18Iv1TP5",
    "user_2RbhDSIMFhBwfr6BluIW2jdX09P",
    "user_2RXktgSvCwR4BCOvKzRJRy9VuIQ",
    "user_2RbeE1lUbTKfQhJlTrIhr2sIehL",
    "user_2RZBsSknclyx2w46SjZA12hzuGK",
    "user_2RVfbm3S3CtPAYMrg5hkbYy8npV",
    "user_2RXMo1QHYCmsw9KMW4FrvAEldic",
    "user_2Rfxkcde7khmanIlFS9hxgwSN52",
    "user_2RnS9wirKFY0srKMCpni42wmbbe",
    "user_2Rjg9CKi8fGglAjrEr6ZD1WdrB4",
    "user_2RUYiizAVzCd1XFJw8iC5Ile9JH",
    "user_2RUWcZayOmGXZAv0VsFGy0GW4ma",
    "user_2Red5XS9wqjYfF4uNCLEHQcr0KD",
    "user_2RWRmnAXzL0fZdNjqxM9glRXehl",
    "user_2RmxSONeSKIrc5CIzYIzYoDdR6v",
    "user_2RW9eAx3uvlmFXpPydlrTlUF85f",
    "user_2Rdo5kZ4EC2KyMYmDaKeso4bxhb",
    "user_2RXIGHuQOc4HmdvtZUecumq8skW",
    "user_2RWwdGFwHhJdBYEhMJ1PM4chRC1",
    "user_2RTyTX9h2ONEr57qxgqmyaV1Oaa",
    "user_2RjK2F1Qds1YDFn6NW2SPzHm2XK",
    "user_2RU6zmWdLaBICO862yONpctw944",
    "user_2RjzTdHJf0wy3kivUXWUIdbxhp8",
    "user_2RUHHjzH9pa2q6Ahxm61f7rITzi",
    "user_2RQZte8vSMVKpb0utdFGhTMq2O1",
    "user_2RWl6ihYUdVbdzdpVftGwGIezTr",
    "user_2RWrULJZNYZeM4mlEDWmrukOyiq",
    "user_2RWS8qjjkEqzhk7KrDwVosc2F0I",
    "user_2RXa4NE7U8wmVfTWCNeL10gSsX8",
    "user_2RY8PLJ2dghQKccnc3dm1lSddQD",
    "user_2RTzqIziB2vL9BfSgl9tLpC88Go",
    "user_2RbmSLXY6nw43OIE6cqW8Aap12S",
    "user_2RXBAbLmzQYFPm5aVP2t7kSqAjZ",
    "user_2Rawjq9bK4y76p1dcJARsoEa9sg",
    "user_2RV5zHLi7kUwHjyK7cgSkIOOckS",
    "user_2RYwgpZLn7GFd6mLAjj9aXXl5vW",
    "user_2RWuKTu3h5eAwGM0GUKdCjf0PYk",
    "user_2RU0UEmVEWcYAT7IWvwWMU0xUqX",
    "user_2RWtYIG1RRhVYyXbUpUaQ70lwnu",
    "user_2RXh0CVZRHMpd2vYZeX4XZTV0PL",
    "user_2Rdyx0gFAIW0P4q1c8uKf34YXAF",
    "user_2RYAkLSg0CewA2sc32ILUO0YmDs",
    "user_2RYR7xqBYXoEJmfq5AjbTmIENUC",
    "user_2RZ43rt7WOFxw019F7aqeoZCg3z",
    "user_2RkFALNktgzhbJNKqVN6UySLGOH",
    "user_2RYHk3oFE89vxLa9fNxKsHYZnY9",
    "user_2RfjgYScMcm6JRtNoFZCqxfcxpu",
    "user_2RcRj5q4EVweBkjgx1OatNKpgFJ",
    "user_2RV9HYjbzrnMn7MAyaA1BWTb807",
    "user_2RVj5fquQR5zWFHk2p8QxaANjXk",
    "user_2RY3fZhm9at4nDVVn39fkPcqiSg",
    "user_2Rk4SdafkXVCWE0Xc6D0oujs16E",
    "user_2RZALBLpVDetKN6Jq2L7DRvjbNP",
    "user_2Rdtu1GacUKM1z0yFb2F65uDbSV",
    "user_2RUaDQd6xHIYmtjnCTIZ0QhQHMK",
    "user_2RYSM9SyQt0f5ENBch8AZmruuke",
    "user_2RWpMR7tBgFf8avP1UIqMsC9nPa",
    "user_2RivgniMGB340ke0HhHIcMyeFJx",
    "user_2RU0Za0KffQxqJ8momd9wykd3qq",
    "user_2RhIlk9J7iIHEsbbP3807CJMkyX",
    "user_2Rb4FTSkmFdHCHSfJEHotWYX834",
    "user_2R7gAtG004yDMDsA74UDwiOmlWs",
    "user_2RjXl7YMkggZXQsGrLN7kM6a4Wi",
    "user_2RUJY8ydSUvIjS4QyCZljeUQg5F",
    "user_2RWkDSLnrJuumD69SHPDPKNxCTW",
    "user_2RWUpheYJ0xH6rpVcMb6xJ1Ek9r",
    "user_2RiUJSSj3bQ1C256en0AYrRvniN",
    "user_2RYinm7cBpenBNtxpD015ISVCFg",
    "user_2RgIgeblB2etpasgYJZgzSTUkHi",
    "user_2Re4NvCgAM0FC4XuAim7x93cesL",
    "user_2RdaX8BrWtxSPn6rlTPm1Zs71fr",
    "user_2RWQcQGDUhWag7EcPjlWN9EdbDW",
    "user_2RYMdqxnXvYpT9And5xDpi1npDK",
    "user_2RYDvNu8lauO8N5OEcyu4HLuOJJ",
    "user_2Raydz7qrRbmHGOWVEqyC6eyCfH",
    "user_2Rgy4k3uWQpMNvJ16eCf6HqGRlT",
    "user_2RXpF4n0KKxqd2PxSBoZShMSarV",
    "user_2RXXpn4RYuY4dmhLK1yaYfIWm1n",
    "user_2RaUOwgL03UeYxvLRhK20YQKbVA",
    "user_2RXQy6IZzgYFggfn7dXzuCglifw",
    "user_2RWXrzET3SMU40AWECqywunq0d1",
    "user_2RVixOflaPpJBckEvo3JH8JmPeO",
    "user_2RiTbYUeFQHiDTxYLolsv3s8JEg",
    "user_2RVqhBtIz8qKXOmNFjpx4Qdsm2y",
    "user_2RTz42z6nM7bh9rLOD6z6tMQmvh",
    "user_2RWNeeefY0PGJMss0ZyE7YIolci",
    "user_2RZ3fOLXcPlyeLBjqfNematybji",
    "user_2RX1XcdpBdC6PvwuI0TWmwzLmUd",
    "user_2RXZrBozYIQC4Tn3QA3rh9gzMDs",
    "user_2RZIMUrD0liVn5GP2b83aLy0LC6",
    "user_2RWmbk31rfUbVsIMwYwTgCM6LWf",
    "user_2RYgG01qDPfqVMnZgPTDmnlkKON",
    "user_2RU0DHgTqVTa4NMVSovAEYUaToy",
    "user_2RWV4y0aNRSoAwKD30xcHMk2Bjr",
    "user_2RlEJVI5bOEA5wPXwNymaaT4WPF",
    "user_2RWjjwGo4m8AGItjfmdM5qYasL8",
    "user_2RWsjdsBjzIUQwun9q5SNiaNlCl",
    "user_2RWW04viPBhNbcr2qlFB2ImWq4r",
  ];

  const res = await fetch("https://api.clerk.com/v1/users?limit=500", {
    headers: {
      Authorization: "Bearer sk_live_Jnk4D6pS7FvW7H1LSd2Z7UI2f5AWP7iCqBqmckwZYA",
    },
  });

  const body = (await res.json()) as { id: string; first_name: string; email_addresses: any }[];

  const emails = body
    .filter((u) => userIds.includes(u.id))
    .map((u) => ({ name: u.first_name, email: u.email_addresses.at(0).email_address }));

  console.log(JSON.stringify(emails, null, 2));
}

main();
