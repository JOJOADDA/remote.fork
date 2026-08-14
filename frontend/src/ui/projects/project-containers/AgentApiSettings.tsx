import { useEffect, useState } from "preact/hooks";
import type { ProjectSecret } from "../../../models/project";
import { Key } from "../../primitives/icons";

const FIELDS = [
  { key: "OPENROUTER_API_KEY", placeholder: "sk-or-…", group: "claude" },
  { key: "OPENAI_API_KEY", placeholder: "API key", group: "openai" },
  { key: "OPENAI_BASE_URL", placeholder: "https://…/v1", group: "openai" },
  { key: "AZURE_OPENAI_API_KEY", placeholder: "Azure key", group: "azure" },
  { key: "AZURE_OPENAI_ENDPOINT", placeholder: "https://resource.openai.azure.com", group: "azure" },
  { key: "AZURE_OPENAI_DEPLOYMENT", placeholder: "deployment name", group: "azure" },
] as const;

type Group = "claude" | "openai" | "azure";
type Field = (typeof FIELDS)[number];

export function AgentApiSettings({ secrets, onSave }: { secrets: ProjectSecret[]; onSave: (key: string, value: string) => Promise<void> }) {
  const [draft, setDraft] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState<Group | null>(null);
  const [saved, setSaved] = useState<Group | null>(null);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    const values: Record<string, string> = {};
    for (const field of FIELDS) values[field.key] = secrets.find((secret) => secret.key === field.key)?.value ?? "";
    setDraft(values);
  }, [secrets]);
  const saveGroup = async (group: Group) => {
    setSaving(group);
    setSaved(null);
    setError(null);
    try {
      for (const field of FIELDS) if (field.group === group && draft[field.key]?.trim()) await onSave(field.key, draft[field.key].trim());
      setSaved(group);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSaving(null);
    }
  };
  return (
    <section class="rounded-md border border-accent-blue/20 bg-accent-blue/[0.04] p-3 space-y-3">
      <div class="flex items-start gap-3"><div class="h-9 w-9 rounded-md bg-accent-blue/[0.12] border border-accent-blue/20 grid place-items-center flex-none text-accent-blue"><Key class="w-4 h-4" /></div><div><div class="text-[14px] font-semibold text-ink-100">Direct API providers</div><p class="text-[12px] text-ink-300 mt-1 leading-relaxed">Configure pay-as-you-go credentials for this project. Values are stored as project secrets and passed only to the selected agent run.</p></div></div>
      <ProviderCard title="Claude via OpenRouter" description="OpenRouter is translated into Claude's API environment automatically." fields={FIELDS.filter((field) => field.group === "claude")} draft={draft} onChange={(key, value) => setDraft((current) => ({ ...current, [key]: value }))} onSave={() => void saveGroup("claude")} saving={saving === "claude"} saved={saved === "claude"} />
      <ProviderCard title="Codex or OpenAI-compatible provider" description="Use OPENAI_BASE_URL for OpenRouter, DeepSeek, or another OpenAI-style endpoint." fields={FIELDS.filter((field) => field.group === "openai")} draft={draft} onChange={(key, value) => setDraft((current) => ({ ...current, [key]: value }))} onSave={() => void saveGroup("openai")} saving={saving === "openai"} saved={saved === "openai"} />
      <ProviderCard title="Azure OpenAI" description="Azure settings configure Codex's Azure Responses API provider." fields={FIELDS.filter((field) => field.group === "azure")} draft={draft} onChange={(key, value) => setDraft((current) => ({ ...current, [key]: value }))} onSave={() => void saveGroup("azure")} saving={saving === "azure"} saved={saved === "azure"} />
      {error && <div class="text-[12px] text-accent-red bg-accent-red/[0.08] border border-accent-red/25 rounded px-2.5 py-2">{error}</div>}
      <p class="text-[11.5px] text-ink-400 leading-relaxed">OAuth remains available in Settings → Agents, but API keys do not require a subscription or browser login.</p>
    </section>
  );
}

function ProviderCard({ title, description, fields, draft, onChange, onSave, saving, saved }: { title: string; description: string; fields: ReadonlyArray<Field>; draft: Record<string, string>; onChange: (key: string, value: string) => void; onSave: () => void; saving: boolean; saved: boolean }) {
  return (
    <div class="rounded-md border border-white/10 bg-white/[0.03] p-3 space-y-2.5">
      <div><div class="text-[13px] font-semibold text-ink-100">{title}</div><div class="text-[11.5px] text-ink-400 mt-0.5 leading-relaxed">{description}</div></div>
      <div class="grid gap-2 md:grid-cols-2">{fields.map((field) => <label key={field.key} class="space-y-1"><span class="block text-[11.5px] text-ink-300 font-mono">{field.key}</span><input type="password" value={draft[field.key] ?? ""} onInput={(event) => onChange(field.key, (event.target as HTMLInputElement).value)} placeholder={field.placeholder} autoComplete="off" spellcheck={false} class="w-full h-9 px-2.5 rounded border border-white/10 bg-black/30 text-[12.5px] font-mono text-ink-50 placeholder-ink-500 focus:outline-none focus:border-accent-blue/50" /></label>)}</div>
      <div class="flex items-center gap-2"><button type="button" onClick={onSave} disabled={saving} class="h-8 px-2.5 rounded bg-accent-blue/80 hover:bg-accent-blue text-white text-[12px] font-medium disabled:opacity-50">{saving ? "Saving…" : "Save provider settings"}</button>{saved && <span class="text-[11.5px] text-accent-green">Saved</span>}</div>
    </div>
  );
}
