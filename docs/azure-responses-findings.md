# Azure Responses API findings

## Sources

1. Microsoft, Codex with Azure OpenAI in Microsoft Foundry Models: https://learn.microsoft.com/en-us/azure/foundry/openai/how-to/codex
2. Microsoft, Use the Azure OpenAI Responses API: https://learn.microsoft.com/en-us/azure/foundry/openai/how-to/responses
3. OpenAI, Migrate to the Responses API: https://developers.openai.com/api/docs/guides/migrate-to-responses

## Key facts

Microsoft's Codex guidance requires the v1 Responses endpoint in `base_url`, an environment-variable name in `env_key`, and `wire_api = "responses"`. The deployment name is passed as `model`.

Microsoft's Responses guidance documents two valid multi-turn approaches: use `previous_response_id`, or manually carry forward `response.output` items and append the next user message. The documented manual approach uses `inputs += response.output`, preserving typed Responses items.

OpenAI's migration guidance explains that Responses uses typed Items rather than Chat Completions messages. An assistant message output contains `content` items with `type: "output_text"` and an `annotations` array. Replaying provider-specific output items or converting them into incomplete generic content blocks can therefore violate the Azure schema.

The user's error occurs after the first successful response at `input[3].content[0].annotations`, strongly indicating malformed replay of a prior Responses item/tool output, not OAuth authentication failure. The service's fresh-turn workaround should avoid native Codex rollout resume, but the remaining error indicates the Codex CLI itself is still replaying an internal Responses state or receiving a malformed tool-context item. The next diagnostic must inspect the effective Codex config, environment precedence, CLI version, and whether Azure is truly being used with `wire_api = "responses"` rather than an OpenRouter-compatible fallback.

## Recommended deployment

For the full Codex agent loop, use an Azure deployment named `gpt-5.3-codex` with the `Responses` API capability and a `Succeeded` provisioning state. Configure the project with `AZURE_OPENAI_ENDPOINT=https://<resource>.services.ai.azure.com/openai/v1`, `AZURE_OPENAI_API_KEY`, and `AZURE_OPENAI_DEPLOYMENT=gpt-5.3-codex`. The deployment name must exactly match the Azure deployment name. `gpt-5.6-terra` remains available for text Responses tests, but `gpt-5.3-codex` is the preferred deployment for Codex file and tool execution.
