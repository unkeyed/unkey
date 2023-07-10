defmodule ElixirMixSupervisionExample do
  require Logger

  @moduledoc """
  Documentation for `ElixirMixSupervisionExample`.
  """

  @doc """
  Hello world.

  ## Examples

      iex> ElixirMixSupervisionExample.hello()
      :world

  """
  def hello do
    :world
  end

  def create_key_via_sdk do
    api_id = Application.get_env(:unkey_elixir_sdk_example, :api_id)

    try do
      expiry =
        DateTime.utc_now()
        |> DateTime.add(100_000)
        |> DateTime.to_unix(:millisecond)

      opts =
        UnkeyElixirSdk.create_key(%{
          "apiId" => api_id,
          "prefix" => "xyz",
          "byteLength" => 16,
          "ownerId" => "glamboyosa",
          "meta" => %{"hello" => "world"},
          "expires" => expiry,
          "ratelimit" => %{
            "type" => "fast",
            "limit" => 10,
            "refillRate" => 1,
            "refillInterval" => 1000
          }
        })

      Logger.info(opts)
      opts
    catch
      err ->
        Logger.error(err)
    end
  end

  def verify_key_via_sdk do
    try do
      opts = create_key_via_sdk()
      is_verified = UnkeyElixirSdk.verify_key(opts["key"])
    catch
      err ->
        Logger.error(err)
    end
  end

  def revoke_key_via_sdk do
    try do
      opts = create_key_via_sdk()
      :ok = UnkeyElixirSdk.revoke_key(opts["keyId"])
    catch
      err ->
        Logger.error(err)
    end
  end
end
