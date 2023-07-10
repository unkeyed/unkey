# Unkey + Elixir Mix Supervision

A simple example of using the [Unkey Elixir SDK](https://github.com/glamboyosa/unkey-elixir-sdk).

## Installation

The package can be installed from Hex PM by adding `unkey_elixir_sdk` to your list of dependencies in `mix.exs`:

> Note: This project uses Elixir version `1.13`.

```elixir
def deps do
  [
    {:unkey_elixir_sdk, "~> 0.1.1"}
  ]
end
```

## Start the GenServer

This example uses a supervision tree to [start the SDK](https://github.com/unkeyed/unkey/blob/135-mix-supervision-tree-example-elixir/examples/elixir_mix_supervision_example/lib/elixir_mix_supervision_example/application.ex)

The GenServer takes a map with two properties.

- token: Your [Unkey](https://unkey.dev) Access token used to make requests. You can create one [here](https://unkey.dev/app/keys) **required**
- base_url: The base URL endpoint you will be hitting i.e. `https://api.unkey.dev/v1/keys` (optional).

```elixir
 children = [
      {UnkeyElixirSdk, %{token: "yourunkeyapitoken"}}
    ]


# Now we start the supervisor with the children and a strategy
{:ok, pid} = Supervisor.start_link(children, strategy: :one_for_one)

# After started, we can query the supervisor for information
Supervisor.count_children(pid)
#=> %{active: 1, specs: 1, supervisors: 0, workers: 1}
```

> **NOTE** In order to run this project either create a `config/dev.secret.exs` file with the key `unkey_elixir_sdk_example` and values specified in `.env.example` ([token](<(https://unkey.dev/app/keys)>) & [apiId](https://docs.unkey.dev/quickstart#4-create-your-first-api-key)) OR directly use the environment variables in code (not recommended).

## Run the project

You can run the project via the IEX console.

```bash
iex -S mix
```

and call the methods like so:

```elixir
ElixirMixSupervisionExample.create_key_via_sdk
```

> The full list of callable methods can be found [here](https://github.com/unkeyed/unkey/blob/135-mix-supervision-tree-example-elixir/examples/elixir_mix_supervision_example/lib/elixir_mix_supervision_example.ex).
