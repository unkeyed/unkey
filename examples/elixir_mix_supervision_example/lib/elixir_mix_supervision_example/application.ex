defmodule ElixirMixSupervisionExample.Application do
  # See https://hexdocs.pm/elixir/Application.html
  # for more information on OTP Applications
  @moduledoc false
  # alias Task.Supervisor

  use Application

  @impl true
  def start(_type, _args) do
    children = [
      {UnkeyElixirSdk, %{token: Application.get_env(:unkey_elixir_sdk_example, :token)}}
    ]

    # See https://hexdocs.pm/elixir/Supervisor.html
    # for other strategies and supported options
    opts = [strategy: :one_for_one, name: ElixirMixSupervisionExample.Supervisor]
    Supervisor.start_link(children, opts)
  end
end
