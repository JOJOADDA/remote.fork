import { useEffect, useRef, useState } from "preact/hooks";
import type { ChatProvider } from "../../../models/chat";
import { modelDisplayLabel } from "../../../config/chat";
import { ChevronDown } from "../../primitives/icons";

export function ComposerModelPicker({
  provider,
  model,
  streaming,
  options,
  onChange,
}: {
  provider: ChatProvider;
  model: string;
  streaming: boolean;
  options: readonly { value: string; label: string; sub: string }[];
  onChange: (model: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState(model);
  const rootRef = useRef<HTMLDivElement>(null);
  const label = modelDisplayLabel(model, provider);

  useEffect(() => {
    setOpen(false);
    setQuery(model);
  }, [provider, model]);

  useEffect(() => {
    if (!open) return;
    function closeOnOutsideClick(event: MouseEvent) {
      const target = event.target as Node | null;
      if (target && !rootRef.current?.contains(target)) setOpen(false);
    }
    window.addEventListener("mousedown", closeOnOutsideClick);
    return () => window.removeEventListener("mousedown", closeOnOutsideClick);
  }, [open]);

  function pick(value: string) {
    setOpen(false);
    setQuery(value);
    if (value !== model) onChange(value);
  }

  const normalizedQuery = query.trim().toLowerCase();
  const filteredOptions = options.filter((option) =>
    !normalizedQuery || `${option.label} ${option.sub} ${option.value}`.toLowerCase().includes(normalizedQuery)
  );

  return (
    <div ref={rootRef} class="relative w-[152px] flex-none sm:w-[168px]">
      <button
        type="button"
        onClick={() => setOpen((value) => !value)}
        class={`h-10 w-full min-w-0 rounded-md px-2.5 text-left transition disabled:cursor-not-allowed disabled:opacity-60 sm:h-7 sm:px-2
                ${open ? "bg-accent-blue/[0.12]" : "bg-white/[0.045] hover:bg-white/[0.075]"}`}
        disabled={streaming}
        title={streaming ? "Cannot change model while streaming" : "Choose model"}
        aria-haspopup="listbox"
        aria-expanded={open}
      >
        <span class="flex min-w-0 items-center gap-1.5">
          <span class="sr-only">Model</span>
          <span class="min-w-0 flex-1 truncate text-[11.5px] font-semibold text-ink-100">{label}</span>
          <ChevronDown class="h-3 w-3 flex-none text-ink-400" />
        </span>
      </button>

      {open && (
        <div
          class="theme-menu-surface fixed inset-x-3 bottom-3 z-50 max-h-[72vh] overflow-y-auto rounded-2xl border border-white/10 bg-[#14161d] p-2 shadow-2xl sm:absolute sm:inset-x-auto sm:bottom-full sm:left-0 sm:mb-2 sm:max-h-none sm:w-[min(23rem,calc(100vw-1.5rem))] sm:rounded-lg sm:p-1"
          role="listbox"
        >
          <div class="border-b border-white/10 p-2">
            <div class="mb-1 flex items-center justify-between gap-2">
              <label class="text-[11px] text-ink-400">Search or enter a model / Azure deployment</label>
              <button type="button" onClick={() => setOpen(false)} class="rounded px-2 py-1 text-xs text-ink-400 hover:bg-white/10 sm:hidden">Close</button>
            </div>
            <div class="flex gap-1.5">
              <input
                type="search"
                value={query}
                onInput={(event) => setQuery((event.target as HTMLInputElement).value)}
                onKeyDown={(event) => {
                  if (event.key === "Enter" && query.trim()) pick(query.trim());
                }}
                placeholder="e.g. DeepSeek-V4-Pro"
                autofocus
                class="min-w-0 flex-1 rounded-lg border border-white/10 bg-black/25 px-3 py-2.5 text-sm text-ink-100 placeholder-ink-500 focus:outline-none focus:border-accent-blue/50 sm:rounded-md sm:px-2 sm:py-1.5 sm:text-xs"
              />
              <button type="button" onClick={() => query.trim() && pick(query.trim())} disabled={!query.trim()} class="rounded-lg bg-accent-blue/80 px-3 text-xs font-semibold text-white disabled:opacity-40 sm:rounded-md sm:px-2">Use</button>
            </div>
          </div>
          {model && !options.some((option) => option.value === model) && (
            <button
              type="button"
              onClick={() => pick(model)}
              class="w-full rounded-md bg-accent-blue/[0.14] px-3 py-2.5 text-left text-accent-blue"
              role="option"
              aria-selected="true"
            >
              <span class="block text-[13px] font-semibold">{model}</span>
              <span class="block text-[12px] text-ink-300">custom model</span>
            </button>
          )}
          {filteredOptions.map((option) => {
            const active = (model || "") === option.value;
            return (
              <button
                key={option.value || "auto"}
                type="button"
                onClick={() => pick(option.value)}
                class={`w-full rounded-md px-3 py-2.5 text-left transition
                        ${active ? "bg-accent-blue/[0.14] text-accent-blue" : "text-ink-100 hover:bg-white/[0.07]"}`}
                role="option"
                aria-selected={active}
              >
                <span class="flex items-center justify-between gap-3">
                  <span class="min-w-0">
                    <span class="block truncate text-[13px] font-semibold">{option.label}</span>
                    <span class="block truncate text-[12px] text-ink-300">{option.sub}</span>
                  </span>
                  {active && <span class="h-2 w-2 flex-none rounded-full bg-accent-blue" />}
                </span>
              </button>
            );
          })}
          {filteredOptions.length === 0 && <div class="px-3 py-4 text-center text-xs text-ink-400">No matching model. Press Use to select this value.</div>}
        </div>
      )}
    </div>
  );
}
