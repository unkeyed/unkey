# Prompt Engineering Guide for Language Models

As an AI language model, your ability to generate high-quality responses depends heavily on the prompts you receive. This guide will help you understand the key principles of effective prompt engineering.

## Executive Summary

This guide provides comprehensive strategies for effective prompt engineering when working with large language models (LLMs). Key points include:

1. Understanding LLMs: Their capabilities, limitations, and how they process input.
2. Importance of prompts: How they shape and guide model outputs.
3. Key principles: Clear instructions, including examples, considering temperature settings, and iterative refinement.
4. Advanced techniques: Command grammars, embedding data, citations, and programmatic consumption of outputs.
5. Practical strategies: Chain-of-thought prompting, handling nested data, and working with various data formats.

By mastering these concepts and techniques, you can significantly improve the quality and reliability of LLM-generated responses for a wide range of applications.

## Understanding Large Language Models (LLMs)

- LLMs are prediction engines that take a sequence of words as input and predict the most likely sequence to follow.
- They assign probabilities to potential next sequences and select one.
- Their knowledge comes from training on massive text corpora, which influences their strengths and limitations.
- While LLM responses often seem plausible, they may sometimes be random or not grounded in reality. This is a problem, always write prompts that are specific and avoid ambiguity or hallucinations.
- When tasking the AI to write prompts, consider these characteristics to craft effective instructions.

## The Importance of Prompts

- Prompts are the starting points for your text generation.
- Crafting your prompt is the means by which you "instruct" or "program" the model.
- They shape your responses and guide you towards the desired output.

## Key Principles of Prompt Engineering

1. Start with Clear Instructions:
   - Begin with a clear, specific instruction about the desired output.
   - Example: "Create a slogan for an organic coffee shop" is more specific than "Create a slogan for a coffee shop."
   - The results for these two prompts are the following:
    - "Fueling your day, one cup at a time."
    - "Fuel your day the natural way with our organic brews!"
   - Note how introducing a single descriptive term to our prompt influences the completion.

2. Include Examples:
   - Demonstrate patterns or subtleties by providing examples in the prompt.
   - This helps convey the expected format and style of the output.
   - Without examples:
     Prompt:

     ```
     Create three slogans for a coffee shop with live music.
     ```

     Completion:

     ```
     1. "Sip, savor, and sway to the rhythm at our coffee shop with live music!"
     2. "Fuel your day with a cup of joe and a side of soulful tunes at our coffee shop."
     3. "Where the coffee is hot and the music is live - come enjoy the vibe!"
     ```

    Evaluation:
    - These slogans are fine, but could be even better
    - It appears the model overlooked the 'live' part in our prompt
   - With examples:
   Prompt:

     ```
     Create three slogans for a business with unique features.
     Business: Bookstore with cats
     Slogans: "Purr-fect Pages", "Books and Whiskers", "Novels and Nuzzles"
     Business: Gym with rock climbing
     Slogans: "Peak Performance", "Reach New Heights", "Climb Your Way Fit"
     Business: Coffee shop with live music
     Slogans:
     ```

     Completion:

     ```
     "Java Jams", "Brews and Beats", "Caffeine and Concerts"
     ```

     Evaluation:
     - Incorporating examples of expected output for a certain input prompted the model to generate the kind of names we aimed for.

- When including examples, comment above that you want me to review the examples and potentially provide placeholders for me to fill in

3. Consider Temperature Settings:
   - Temperature (0 to 1) controls the randomness of your outputs.
   - Lower temperature (0) leads to more deterministic, focused responses.
   - Higher temperature (1) produces more diverse, creative outputs.
   - Choose based on the task: lower for factual tasks, higher for creative tasks.
   - Comment on the nature of the task you're prompting for to guide the model's creativity & adjust the temperature accordingly.
   - Example with temperature 1:
     Prompt:

     ```
     Create three slogans for a business with unique features.
     Business: Bookstore with cats
     Slogans: "Purr-fect Pages", "Books and Whiskers", "Novels and Nuzzles"
     Business: Gym with rock climbing
     Slogans: "Peak Performance", "Reach New Heights", "Climb Your Way Fit"
     Business: Coffee shop with live music
     Slogans:
     ```

     Completion:
     - first completion:

     ```
     "Beans and Beats", "Espresso and Echoes", "Music, Mocha, and More"
     ```

     - second completion:

     ```
     "Jazz up your Java", "Sip, Sit, and Listen", "Groove and Grind"
     ```

     - third completion:

     ```
     "Java Jams", "A Sip of Sound", "Caffeine and Concerts"
     ```

     - fourth completion:

     ```
     "Beans and Beats", "Caffeine and Concerts", "Music to Your Mocha"
     ```

     Evaluation:
     - With a temperature above 0, the same prompt delivers varied completions each time.
     - Keep in mind that the model forecasts the text most likely to follow the preceding text.

4. Iterate and Refine:
   - If the initial output isn't satisfactory, refine the prompt and try again.
   - Add more context, examples, or specific instructions as needed.

### Teach a Bot to Fish

Sometimes you’ll want the bot to have the capability to perform actions on the user’s behalf, like adding a memo to a receipt or plotting a chart. Or perhaps we want it to retrieve data in more nuanced ways than semantic search would allow for, like retrieving the past 90 days of expenses.

In these scenarios, we need to teach the bot how to fish.

#### Command Grammars

We can give the bot a list of commands for our system to interpret, along with descriptions and examples for the commands, and then have it produce programs composed of those commands.

There are many caveats to consider when going with this approach. With complex command grammars, the bot will tend to hallucinate commands or arguments that could plausibly exist, but don’t actually. The art to getting this right is enumerating commands that have relatively high levels of abstraction, while giving the bot sufficient flexibility to compose them in novel and useful ways.

For example, giving the bot a `plot-the-last-90-days-of-expenses` command is not particularly flexible or composable in what the bot can do with it. Similarly, a `draw-pixel-at-x-y [x] [y] [rgb]` command would be far too low-level. But giving the bot a `plot-expenses` and `list-expenses` command provides some good primitives that the bot has some flexibility with.

In an example below, we use this list of commands:

| Command            | Arguments                   | Description                                      |
| ------------------ | --------------------------- | ------------------------------------------------ |
| list-expenses      | budget                      | Returns a list of expenses for a given budget    |
| converse           | message                     | A message to show to the user                    |
| plot-expenses      | expenses[]                  | Plots a list of expenses                         |
| get-budget-by-name | budget_name                 | Retrieves a budget by name                       |
| list-budgets       |                             | Returns a list of budgets the user has access to |
| add-memo           | inbox_item_id, memo message | Adds a memo to the provided inbox item           |

We provide this table to the model in Markdown format, which the language model handles incredibly well – presumably because OpenAI trains heavily on data from GitHub.

In this example below, we ask the model to output the commands in [reverse polish notation](https://en.wikipedia.org/wiki/Reverse_Polish_notation)[^7].

[^7]: The model handles the simplicity of RPN astoundingly well.

<p align="center">
  <img width="550" src="https://user-images.githubusercontent.com/89960/233505150-aef4409c-03ba-4669-95d7-6c48f3c2c3ea.png" title="A bot happily generating commands to run in response to user queries.">
</p>

> 🧠 There are some interesting subtle things going on in that example, beyond just command generation. When we ask it to add a memo to the “shake shack” expense, the model knows that the command `add-memo` takes an expense ID. But we never tell it the expense ID, so it looks up “Shake Shack” in the table of expenses we provided it, then grabs the ID from the corresponding ID column, and then uses that as an argument to `add-memo`.

Getting command grammars working reliably in complex situations can be tricky. The best levers we have here are to provide lots of descriptions, and as **many examples** of usage as we can. Large language models are [few-shot learners](https://en.wikipedia.org/wiki/Few-shot_learning_(natural_language_processing)), meaning that they can learn a new task by being provided just a few examples. In general, the more examples you provide the better off you’ll be – but that also eats into your token budget, so it’s a balance.

Here’s a more complex example, with the output specified in JSON instead of RPN. And we use Typescript to define the return types of commands.

<p align="center">
  <img width="550" src="https://user-images.githubusercontent.com/89960/233505696-fc440931-9baf-4d06-80e7-54801532d63f.png" title="A bot happily generating commands to run in response to user queries.">
</p>

<details>

  <summary>(Full prompt)</summary>
  
~~~
You are a financial assistant working at Brex, but you are also an expert programmer.

I am a customer of Brex.

You are to answer my questions by composing a series of commands.

The output types are:

```typescript
type LinkedAccount = {
    id: string,
    bank_details: {
        name: string,
        type: string,
    },
    brex_account_id: string,
    last_four: string,
    available_balance: {
        amount: number,
        as_of_date: Date,
    },
    current_balance: {
            amount: number,
        as_of_date: Date,
    },
}

type Expense = {
  id: string,
  memo: string,
  amount: number,
}

type Budget = {
  id: string,
  name: string,
  description: string,
  limit: {
    amount: number,
    currency: string,
  }
}
```

The commands you have available are:

| Command              | Arguments           | Description                                                                               | Output Format   |
| -------------------- | ------------------- | ----------------------------------------------------------------------------------------- | --------------- |
| nth                  | index, values[]     | Return the nth item from an array                                                         | any             |
| push                 | value               | Adds a value to the stack to be consumed by a future command                              | any             |
| value                | key, object         | Returns the value associated with a key                                                   | any             |
| values               | key, object[]       | Returns an array of values pulled from the corresponding key in array of objects          | any[]           |
| sum                  | value[]             | Sums an array of numbers                                                                  | number          |
| plot                 | title, values[]     | Plots the set of values in a chart with the given title                                   | Plot            |
| list-linked-accounts |                     | "Lists all bank connections that are eligible to make ACH transfers to Brex cash account" | LinkedAccount[] |
| list-expenses        | budget_id           | Given a budget id, returns the list of expenses for it                                    | Expense[]       |
| get-budget-by-name   | name                | Given a name, returns the budget                                                          | Budget          |
| add-memo             | expense_id, message | Adds a memo to an expense                                                                 | bool            |
| converse             | message             | Send the user a message                                                                   | null            |

Only respond with commands.

Output the commands in JSON as an abstract syntax tree.

IMPORTANT - Only respond with a program. Do not respond with any text that isn't part of a program. Do not write prose, even if instructed. Do not explain yourself.

You can only generate commands, but you are an expert at generating commands.
~~~

</details>

This version is a bit easier to parse and interpret if your language of choice has a `JSON.parse` function.

> 🧙‍♂️ There is no industry established best format for defining a DSL for the model to generate programs. So consider this an area of active research. You will bump into limits. And as we overcome these limits, we may discover more optimal ways of defining commands.

#### ReAct

In March of 2023, Princeton and Google released a paper “[ReAct: Synergizing Reasoning and Acting in Language Models](https://arxiv.org/pdf/2210.03629.pdf)”, where they introduce a variant of command grammars that allows for fully autonomous interactive execution of actions and retrieval of data.

The model is instructed to return a `thought` and an `action` that it would like to perform. Another agent (e.g. our client) then performs the `action` and returns it to the model as an `observation`. The model will then loop to return more thoughts and actions until it returns an `answer`.

This is an incredibly powerful technique, effectively allowing the bot to be its own research assistant and possibly take actions on behalf of the user. Combined with a powerful command grammar, the bot should rapidly be able to answer a massive set of user requests.

In this example, we give the model a small set of commands related to getting employee data and searching wikipedia:

| Command       | Arguments   | Description                                                                               |
| ------------- | ----------- | ----------------------------------------------------------------------------------------- |
| find_employee | name        | Retrieves an employee by name                                                             |
| get_employee  | id          | Retrieves an employee by ID                                                               |
| get_location  | id          | Retrieves a location by ID                                                                |
| get_reports   | employee_id | Retrieves a list of employee ids that report to the employee associated with employee_id. |
| wikipedia     | article     | Retrieves a wikipedia article on a topic.                                                 |

We then ask the bot a simple question, “Is my manager famous?”.

We see that the bot:

1. First looks up our employee profile.
2. From our profile, gets our manager’s id and looks up their profile.
3. Extracts our manager’s name and searches for them on Wikipedia.
    - I chose a fictional character for the manager in this scenario.
4. The bot reads the wikipedia article and concludes that can’t be my manager since it is a fictional character.
5. The bot then modifies its search to include (real person).
6. Seeing that there are no results, the bot concludes that my manager is not famous.

| ![image](https://user-images.githubusercontent.com/89960/233506839-5c8b2d77-1d78-464d-bc33-a725e12f2624.png) | ![image](https://user-images.githubusercontent.com/89960/233506870-05fc415d-efa2-48b7-aad9-b5035e535e6d.png) |
| ------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------ |

<details>
<summary>(Full prompt)</summary>

~~~
You are a helpful assistant. You run in a loop, seeking additional information to answer a user's question until you are able to answer the question.

Today is June 1, 2025. My name is Fabian Seacaster. My employee ID is 82442.

The commands to seek information are:

| Command       | Arguments   | Description                                                                                 |
| ------------- | ----------- | ------------------------------------------------------------------------------------------- |
| find_employee | name        | Retrieves an employee by name                                                               |
| get_employee  | id          | Retrieves an employee by ID                                                                 |
| get_location  | id          | Retrieves a location by ID                                                                  |
| get_reports   | employee_id | Retrieves a list of employee ids that report to the employee associated with `employee_id`. |
| wikipedia     | article     | Retrieves a wikipedia article on a topic.                                                   |

Your response will be in JSON and will include a "Thought" + "Action" to retrieve data that you need in order to answer the question, or it will include the "Answer". When data has been retrieved, it will be included as an "Observation".

You will continue generating thoughts and actions until you get to an answer, or conclude that you can't.

Example 1:
```
User: What is the population of Philadelphia?

Assistant: {
  "thought": "Wikipedia likely has this information. I'm looking it up...",
  "action": {"command": "wikipedia", "article": "Philadelphia"}
}

Assistant: {
  "observation": "Philadelphia, often called Philly, is the largest city in the Commonwealth of Pennsylvania and the second-largest city in both the Northeast megalopolis and Mid-Atlantic regions after New York City. It is one of the most historically significant cities in the United States and served as the nation's capital city until 1800. Philadelphia is the nation's sixth-largest city with a population of 1,603,797 as of the 2020 census. Since 1854, the city has been coextensive with Philadelphia County, the most populous county in Pennsylvania and the urban core of the Delaware Valley, the nation's seventh-largest and one of the world's largest metropolitan regions with 6.245 million residents in 2020. Philadelphia is known for its extensive contributions to American history and for its role in the life sciences, business and industry, art, literature, and music."
}

Assistant: {
  "answer": "The population of Philadelphia, as of the 2020 census, is 1,603,797 people."
}
```

Example 2:
```
User: What is my manager's city?

Assistant: {
  "thought": "Let me look up who your manager is. First, I need to look up your profile.",
  "action": {"command": "get_employee", "id": 92352}
}

Assistant: {
  "observation": {
    "id": 78334,
    "name": "Ms. Manager",
    "location_id": 8832
  }
}

Assistant: {
  "thought": "Your manager is Ms. Manager. I'm looking up their location.",
  "action": {"command": "get_location", "id": 8832}
}

Assistant: {
  "observation": {
    "id": 8832,
    "name": "Philadelphia"
  }
}

Assistant: {
  "answer": "Your manager lives in Philadelphia."
}
```
~~~

</details>

## Further Strategies

This section contains examples and strategies for specific needs or problems. For successful prompt engineering, you will need to combine some subset of all of the strategies enumerated in this document. Don’t be afraid to mix and match things – or invent your own approaches.

### Embedding Data

In hidden contexts, you’ll frequently want to embed all sorts of data. The specific strategy will vary depending on the type and quantity of data you are embedding.

#### Simple Lists

For one-off objects, enumerating fields + values in a normal bulleted list works pretty well:

<p align="center">
  <img width="550" src="https://user-images.githubusercontent.com/89960/233507156-0bdbc0af-d977-44e0-a8d5-b30538c5bbd9.png" title="GPT-4 extracting Steve’s occupation from a list attributes.">
</p>

It will also work for larger sets of things, but there are other formats for lists of data that GPT handles more reliably. Regardless, here’s an example:

<p align="center">
  <img width="550" src="https://user-images.githubusercontent.com/89960/233507223-9cda591e-62f3-4339-b227-a07c37b90724.png" title="GPT-4 answering questions about a set of expenses.">
</p>

#### Markdown Tables

Markdown tables are great for scenarios where you have many items of the same type to enumerate.

Fortunately, OpenAI’s models are exceptionally good at working with Markdown tables (presumably from the tons of GitHub data they’ve trained on).

We can reframe the above using Markdown tables instead:

<p align="center">
  <img width="550" src="https://user-images.githubusercontent.com/89960/233507313-7ccd825c-71b9-46d3-80c9-30bf97a8e090.png" title="GPT-4 answering questions about a set of expenses from a Markdown table.">
</p>

<p align="center">
  <img width="550" src="https://user-images.githubusercontent.com/89960/233507395-b8ecb641-726c-4e57-b85e-13f6b7717f22.png" title="GPT-4 answering questions about a set of expenses from a Markdown table.">
</p>

> 🧠 Note that in this last example, the items in the table have an explicit date, February 2nd. In our question, we asked about “today”. And earlier in the prompt we mentioned that today was Feb 2. The model correctly handled the transitive inference – converting “today” to “February 2nd” and then looking up “February 2nd” in the table.

#### JSON

Markdown tables work really well for many use cases and should be preferred due to their density and ability for the model to handle them reliably, but you may run into scenarios where you have many columns and the model struggles with it or every item has some custom attributes and it doesn’t make sense to have dozens of columns of empty data.

In these scenarios, JSON is another format that the model handles really well. The close proximity of `keys` to their `values` makes it easy for the model to keep the mapping straight.

Here is the same example from the Markdown table, but with JSON instead:

<p align="center">
  <img width="550" src="https://user-images.githubusercontent.com/89960/233507559-26e6615d-4896-4a2c-b6ff-44cbd7d349dc.png" title="GPT-4 answering questions about a set of expenses from a JSON blob.">
</p>

#### Freeform Text

Occasionally you’ll want to include freeform text in a prompt that you would like to delineate from the rest of the prompt – such as embedding a document for the bot to reference. In these scenarios, surrounding the document with triple backticks, ```, works well[^8].

<p align="center">
  <img width="550" src="https://user-images.githubusercontent.com/89960/233507684-93222728-e216-47b4-8554-04acf9ec6201.png" title="GPT-4 answering questions about a set of expenses from a JSON blob.">
</p>

[^8]: A good rule of thumb for anything you’re doing in prompts is to lean heavily on things the model would have learned from GitHub.

#### Nested Data

Not all data is flat and linear. Sometimes you’ll need to embed data that is nested or has relations to other data. In these scenarios, lean on `JSON`:

<p align="center">
  <img width="550" src="https://user-images.githubusercontent.com/89960/233507758-7baffcaa-647b-4869-9cfb-a7cf8849c453.png" title="GPT-4 handles nested JSON very reliably.">
</p>

<details>
<summary>(Full prompt)</summary>

~~~
You are a helpful assistant. You answer questions about users. Here is what you know about them:

{
  "users": [
    {
      "id": 1,
      "name": "John Doe",
      "contact": {
        "address": {
          "street": "123 Main St",
          "city": "Anytown",
          "state": "CA",
          "zip": "12345"
        },
        "phone": "555-555-1234",
        "email": "johndoe@example.com"
      }
    },
    {
      "id": 2,
      "name": "Jane Smith",
      "contact": {
        "address": {
          "street": "456 Elm St",
          "city": "Sometown",
          "state": "TX",
          "zip": "54321"
        },
        "phone": "555-555-5678",
        "email": "janesmith@example.com"
      }
    },
    {
      "id": 3,
      "name": "Alice Johnson",
      "contact": {
        "address": {
          "street": "789 Oak St",
          "city": "Othertown",
          "state": "NY",
          "zip": "67890"
        },
        "phone": "555-555-2468",
        "email": "alicejohnson@example.com"
      }
    },
    {
      "id": 4,
      "name": "Bob Williams",
      "contact": {
        "address": {
          "street": "135 Maple St",
          "city": "Thistown",
          "state": "FL",
          "zip": "98765"
        },
        "phone": "555-555-8642",
        "email": "bobwilliams@example.com"
      }
    },
    {
      "id": 5,
      "name": "Charlie Brown",
      "contact": {
        "address": {
          "street": "246 Pine St",
          "city": "Thatstown",
          "state": "WA",
          "zip": "86420"
        },
        "phone": "555-555-7531",
        "email": "charliebrown@example.com"
      }
    },
    {
      "id": 6,
      "name": "Diane Davis",
      "contact": {
        "address": {
          "street": "369 Willow St",
          "city": "Sumtown",
          "state": "CO",
          "zip": "15980"
        },
        "phone": "555-555-9512",
        "email": "dianedavis@example.com"
      }
    },
    {
      "id": 7,
      "name": "Edward Martinez",
      "contact": {
        "address": {
          "street": "482 Aspen St",
          "city": "Newtown",
          "state": "MI",
          "zip": "35742"
        },
        "phone": "555-555-6813",
        "email": "edwardmartinez@example.com"
      }
    },
    {
      "id": 8,
      "name": "Fiona Taylor",
      "contact": {
        "address": {
          "street": "531 Birch St",
          "city": "Oldtown",
          "state": "OH",
          "zip": "85249"
        },
        "phone": "555-555-4268",
        "email": "fionataylor@example.com"
      }
    },
    {
      "id": 9,
      "name": "George Thompson",
      "contact": {
        "address": {
          "street": "678 Cedar St",
          "city": "Nexttown",
          "state": "GA",
          "zip": "74125"
        },
        "phone": "555-555-3142",
        "email": "georgethompson@example.com"
      }
    },
    {
      "id": 10,
      "name": "Helen White",
      "contact": {
        "address": {
          "street": "852 Spruce St",
          "city": "Lasttown",
          "state": "VA",
          "zip": "96321"
        },
        "phone": "555-555-7890",
        "email": "helenwhite@example.com"
      }
    }
  ]
}
~~~

</details>

If using nested `JSON` winds up being too verbose for your token budget, fallback to `relational tables` defined with `Markdown`:

<p align="center">
  <img width="550" src="https://user-images.githubusercontent.com/89960/233507968-a378587b-e468-4882-a1e8-678d9f3933d3.png" title="GPT-4 handles relational tables pretty reliably too.">
</p>

<details>
<summary>(Full prompt)</summary>

~~~
You are a helpful assistant. You answer questions about users. Here is what you know about them:

Table 1: users
| id (PK) | name            |
| ------- | --------------- |
| 1       | John Doe        |
| 2       | Jane Smith      |
| 3       | Alice Johnson   |
| 4       | Bob Williams    |
| 5       | Charlie Brown   |
| 6       | Diane Davis     |
| 7       | Edward Martinez |
| 8       | Fiona Taylor    |
| 9       | George Thompson |
| 10      | Helen White     |

Table 2: addresses
| id (PK) | user_id (FK) | street        | city      | state | zip   |
| ------- | ------------ | ------------- | --------- | ----- | ----- |
| 1       | 1            | 123 Main St   | Anytown   | CA    | 12345 |
| 2       | 2            | 456 Elm St    | Sometown  | TX    | 54321 |
| 3       | 3            | 789 Oak St    | Othertown | NY    | 67890 |
| 4       | 4            | 135 Maple St  | Thistown  | FL    | 98765 |
| 5       | 5            | 246 Pine St   | Thatstown | WA    | 86420 |
| 6       | 6            | 369 Willow St | Sumtown   | CO    | 15980 |
| 7       | 7            | 482 Aspen St  | Newtown   | MI    | 35742 |
| 8       | 8            | 531 Birch St  | Oldtown   | OH    | 85249 |
| 9       | 9            | 678 Cedar St  | Nexttown  | GA    | 74125 |
| 10      | 10           | 852 Spruce St | Lasttown  | VA    | 96321 |

Table 3: phone_numbers
| id (PK) | user_id (FK) | phone        |
| ------- | ------------ | ------------ |
| 1       | 1            | 555-555-1234 |
| 2       | 2            | 555-555-5678 |
| 3       | 3            | 555-555-2468 |
| 4       | 4            | 555-555-8642 |
| 5       | 5            | 555-555-7531 |
| 6       | 6            | 555-555-9512 |
| 7       | 7            | 555-555-6813 |
| 8       | 8            | 555-555-4268 |
| 9       | 9            | 555-555-3142 |
| 10      | 10           | 555-555-7890 |

Table 4: emails
| id (PK) | user_id (FK) | email                      |
| ------- | ------------ | -------------------------- |
| 1       | 1            | johndoe@example.com        |
| 2       | 2            | janesmith@example.com      |
| 3       | 3            | alicejohnson@example.com   |
| 4       | 4            | bobwilliams@example.com    |
| 5       | 5            | charliebrown@example.com   |
| 6       | 6            | dianedavis@example.com     |
| 7       | 7            | edwardmartinez@example.com |
| 8       | 8            | fionataylor@example.com    |
| 9       | 9            | georgethompson@example.com |
| 10      | 10           | helenwhite@example.com     |

Table 5: cities
| id (PK) | name      | state | population | median_income |
| ------- | --------- | ----- | ---------- | ------------- |
| 1       | Anytown   | CA    | 50,000     | $70,000       |
| 2       | Sometown  | TX    | 100,000    | $60,000       |
| 3       | Othertown | NY    | 25,000     | $80,000       |
| 4       | Thistown  | FL    | 75,000     | $65,000       |
| 5       | Thatstown | WA    | 40,000     | $75,000       |
| 6       | Sumtown   | CO    | 20,000     | $85,000       |
| 7       | Newtown   | MI    | 60,000     | $55,000       |
| 8       | Oldtown   | OH    | 30,000     | $70,000       |
| 9       | Nexttown  | GA    | 15,000     | $90,000       |
| 10      | Lasttown  | VA    | 10,000     | $100,000      |
~~~

</details>

> 🧠 The model works well with data in [3rd normal form](https://en.wikipedia.org/wiki/Third_normal_form), but may struggle with too many joins. In experiments, it seems to do okay with at least three levels of nested joins. In the example above the model successfully joins from `users` to `addresses` to `cities` to infer the likely income for George – $90,000.

### Citations

Frequently, a natural language response isn’t sufficient on its own and you’ll want the model’s output to cite where it is getting data from.

One useful thing to note here is that anything you might want to cite should have a unique ID. The simplest approach is to just ask the model to link to anything it references:

<p align="center">
  <img width="550" src="https://user-images.githubusercontent.com/89960/233509069-1dcbffa2-8357-49b5-be43-9791f93bd0f8.png" title="GPT-4 will reliably link to data if you ask it to.">
</p>

### Programmatic Consumption

By default, language models output natural language text, but frequently we need to interact with this result in a programmatic way that goes beyond simply printing it out on screen. You can achieve this by  asking the model to output the results in your favorite serialization format (JSON and YAML seem to work best).

Make sure you give the model an example of the output format you’d like. Building on our previous travel example above, we can augment our prompt to tell it:

~~~
Produce your output as JSON. The format should be:
```
{
    message: "The message to show the user",
    hotelId: 432,
    flightId: 831
}
```

Do not include the IDs in your message.
~~~

And now we’ll get interactions like this:

<p align="center">
  <img width="550" src="https://user-images.githubusercontent.com/89960/233509174-be0c3bc5-08e3-4d1a-8841-52c401def770.png" title="GPT-4 providing travel recommendations in an easy to work with format.">
</p>

You could imagine the UI for this rendering the message as normal text, but then also adding discrete buttons for booking the flight + hotel, or auto-filling a form for the user.

As another example, let’s build on the [citations](#citations) example – but move beyond Markdown links. We can ask it to produce JSON with a normal message along with a list of items used in the creation of that message. In this scenario you won’t know exactly where in the message the citations were leveraged, but you’ll know that they were used somewhere.

<p align="center">
  <img width="550" src="https://user-images.githubusercontent.com/89960/233509280-59d9ff46-0e95-488a-b314-a7d2b7c9bfa3.png" title="Asking the model to provide a list of citations is a reliable way to programmatically know what data the model leaned on in its response.">
</p>

> 🧠 Interestingly, in the model’s response to “How much did I spend at Target?” it provides a single value, $188.16, but **importantly** in the `citations` array it lists the individual expenses that it used to compute that value.

### Chain of Thought

Sometimes you will bang your head on a prompt trying to get the model to output reliable results, but, no matter what you do, it just won’t work. This will frequently happen when the bot’s final output requires intermediate thinking, but you ask the bot only for the output and nothing else.

The answer may surprise you: ask the bot to show its work. In October 2022, Google released a paper “[Chain-of-Thought Prompting Elicits Reasoning in Large Language Models](https://arxiv.org/pdf/2201.11903.pdf)” where they showed that if, in your hidden prompt, you give the bot examples of answering questions by showing your work, then when you ask the bot to answer something it will show its work and produce more reliable answers.

Just a few weeks after that paper was published, at the end of October 2022, the University of Tokyo and Google released the paper “[Large Language Models are Zero-Shot Reasoners](https://openreview.net/pdf?id=e2TBb5y0yFf)”, where they show that you don’t even need to provide examples – **you simply have to ask the bot to think step-by-step**.

#### Averaging

Here is an example where we ask the bot to compute the average expense, excluding Target. The actual answer is $136.77 and the bot almost gets it correct with $136.43.

<p align="center">
  <img width="550" src="https://user-images.githubusercontent.com/89960/233509534-2b32c8dd-a1ee-42ea-82fb-4f84cfe7e9ba.png" title="The model **almost** gets the average correct, but is a few cents off.">
</p>

If we simply add “Let’s think step-by-step”, the model gets the correct answer:

<p align="center">
  <img width="550" src="https://user-images.githubusercontent.com/89960/233509608-6e53995b-668b-47f6-9b5e-67afad89f8bc.png" title="When we ask the model to show its work, it gets the correct answer.">
</p>

#### Interpreting Code

Let’s revisit the Python example from earlier and apply chain-of-thought prompting to our question. As a reminder, when we asked the bot to evaluate the Python code it gets it slightly wrong. The correct answer is `Hello, Brex!!Brex!!Brex!!!` but the bot gets confused about the number of !'s to include. In below’s example, it outputs `Hello, Brex!!!Brex!!!Brex!!!`:

<p align="center">
  <img width="550" src="https://user-images.githubusercontent.com/89960/233509724-8f3302f8-59eb-4d3b-8939-53d7f63b0299.png" title="The bot almost interprets the Python code correctly, but is a little off.">
</p>

If we ask the bot to show its work, then it gets the correct answer:

<p align="center">
  <img width="550" src="https://user-images.githubusercontent.com/89960/233509790-2a0f2189-d864-4d27-aacb-cfc936fad907.png" title="The bot correctly interprets the Python code if you ask it to show its work.">
</p>

#### Delimiters

In many scenarios, you may not want to show the end user all of the bot’s thinking and instead just want to show the final answer. You can ask the bot to delineate the final answer from its thinking. There are many ways to do this, but let’s use JSON to make it easy to parse:

<p align="center">
  <img width="550" src="https://user-images.githubusercontent.com/89960/233509865-4f3e7265-6645-4d43-8644-ecac5c0ca4a7.png" title="The bot showing its work while also delimiting the final answer for easy extraction.">
</p>

Using Chain-of-Thought prompting will consume more tokens, resulting in increased price and latency, but the results are noticeably more reliable for many scenarios. It’s a valuable tool to use when you need the bot to do something complex and as reliably as possible.

## Prompt Guidelines

1. Prompt Length:
• Aim for concise prompts when possible, but don't sacrifice clarity for brevity.
• For simple tasks, 1-3 sentences are often sufficient.
• For more complex tasks, longer prompts (up to a few paragraphs) may be necessary.
• Be aware of token limits for the specific model you're using.
Complexity:
• Match the complexity of your prompt to the task at hand.
• Use simpler language for straightforward tasks.
• For complex tasks, break them down into smaller steps or use chain-of-thought prompting.
Context:
• Include relevant context, but avoid unnecessary information that might confuse the model.
• If using examples, start with 2-3 and add more if needed for improved performance.
Specificity:
• Be as specific as possible about the desired output format and content.
• Use clear, unambiguous language to reduce the chance of misinterpretation.
Iterative Refinement:
• Start with a simple prompt and iteratively refine it based on the model's outputs.
• Test different variations to find the most effective formulation.
6. Model Considerations:
• Larger models can generally handle longer and more complex prompts effectively.
• Smaller models may require more concise and focused prompts.
