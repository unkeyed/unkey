defmodule ElixirMixSupervisionExample.MixProject do
  use Mix.Project

  def project do
    [
      app: :elixir_mix_supervision_example,
      version: "0.1.0",
      elixir: "~> 1.13",
      start_permanent: Mix.env() == :prod,
      deps: deps()
    ]
  end

  # Run "mix help compile.app" to learn about applications.
  def application do
    [
      extra_applications: [:logger],
      mod: {ElixirMixSupervisionExample.Application, []}
    ]
  end

  # Run "mix help deps" to learn about dependencies.
  defp deps do
    [
      {:unkey_elixir_sdk, "~> 0.1.1"}
    ]
  end
end
