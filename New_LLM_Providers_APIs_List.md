Here is a new markdown list pairing provider names with their OpenAI-compatible base URLs for use with OpenCode.

Provider Name OpenAI-Compatible Base URL Note
Poe `https://api.poe.com/v1`[citation:2]  Use the @ai-sdk/openai-compatible adapter in your OpenCode config.
Moonshot AI (Kimi) `https://api.moonshot.ai/v1`[citation:5]  Configured with the @ai-sdk/openai-compatible adapter.
CBorg https://api.cborg.lbl.gov Also uses the @ai-sdk/openai-compatible adapter.
NaviGator AI `https://api.ai.it.ufl.edu/v1`[citation:9]  A specific example using the mistral-small-3.1 model.
Docker Model Runner (DMR) `http://localhost:12434/engines/v1`[citation:6]  For local, self-hosted models. Use http://model-runner.docker.internal/engines/v1 if running OpenCode in a container.
llama.cpp `http://localhost:8080/v1`[citation:1]  For local models via llama-server (example port). Base URL must point to your local server.
LM Studio `http://localhost:1234/v1`[citation:1]  For local models via LM Studio (example port). Base URL must point to your local server.

⚙️ How to Configure a Provider in OpenCode

Here is the general method to add any provider from the list above to your OpenCode configuration:

1. Add the provider to your ~/.config/opencode/opencode.json file using the format below.
2. Add your API key to ~/.local/share/opencode/auth.json for the corresponding provider id.
3. Select the model within OpenCode using the /models command.

Configuration Format:

```json
{
  "$schema": "https://opencode.ai/config.json",
  "provider": {
    "your_provider_id": {
      "npm": "@ai-sdk/openai-compatible",
      "name": "Your Provider Display Name",
      "options": {
        "baseURL": "https://api.example.com/v1"
      },
      "models": {
        "example-model-id": {
          "name": "Example Model Display Name"
        }
      }
    }
  }
}
```

💡 Key Considerations

· Any OpenAI-Compatible API: This method works for any service that provides an OpenAI-compatible endpoint.
· Authentication: For most cloud providers, you'll need to get an API key from their console and add it to the auth.json file.
· Local Servers: For llama.cpp or LM Studio, ensure the local inference server is running before starting OpenCode.

If you are interested in setting up a specific provider from our earlier list or need help with the configuration files, feel free to ask.
