import type { ChatMessageBlock } from "../../../models/chatMessage";

type StageState = "done" | "active" | "queued";

function Stage({ state, label, detail }: { state: StageState; label: string; detail?: string }) {
  const marker = state === "done" ? "✓" : state === "active" ? "•" : "";
  return (
    <div class="flex min-w-0 items-center gap-2 text-[12px]">
      <span
        class={`grid h-5 w-5 flex-none place-items-center rounded-full border text-[11px] ${
          state === "done"
            ? "border-accent-green/50 bg-accent-green/10 text-accent-green"
            : state === "active"
              ? "border-accent-blue/60 bg-accent-blue/15 text-accent-blue animate-pulse"
              : "border-white/10 text-ink-500"
        }`}
        aria-hidden="true"
      >
        {marker}
      </span>
      <span class={state === "queued" ? "text-ink-500" : "text-ink-200"}>{label}</span>
      {detail && <span class="ml-auto truncate text-[11px] text-ink-500">{detail}</span>}
    </div>
  );
}

export function LiveProgress({ blocks }: { blocks: ChatMessageBlock[] }) {
  const assistant = [...blocks].reverse().find((block) => block.type === "assistant");
  const parts = assistant?.type === "assistant" ? assistant.parts : [];
  const tools = parts.filter((part) => part.kind === "tool");
  const activeTool = tools.find((part) => part.kind === "tool" && part.status === "running");
  const hasThinking = parts.some((part) => part.kind === "thinking");
  const hasText = parts.some((part) => part.kind === "text");
  const hasTools = tools.length > 0;

  return (
    <section
      class="mx-auto max-w-3xl rounded-lg border border-accent-blue/20 bg-accent-blue/[0.06] px-3 py-2.5 shadow-sm"
      role="status"
      aria-live="polite"
      aria-label="Live agent progress"
    >
      <div class="mb-2 flex items-center justify-between gap-2">
        <div class="flex min-w-0 items-center gap-2">
          <span class="h-2 w-2 flex-none rounded-full bg-accent-blue animate-pulse" aria-hidden="true" />
          <span class="truncate text-[12px] font-medium text-ink-100">Live progress</span>
        </div>
        <span class="text-[11px] text-ink-500">Updates as activity arrives</span>
      </div>
      <div class="space-y-1.5">
        <Stage state={hasThinking || hasText || hasTools ? "done" : "active"} label="Understand request" />
        <Stage
          state={hasThinking ? "done" : hasTools || hasText ? "done" : "active"}
          label="Reason and plan"
        />
        <Stage
          state={activeTool ? "active" : hasTools ? "done" : "queued"}
          label="Use tools"
          detail={activeTool?.kind === "tool" ? activeTool.name : hasTools ? `${tools.length} completed` : undefined}
        />
        <Stage state={hasText && !activeTool ? "active" : "queued"} label="Prepare response" />
      </div>
    </section>
  );
}

export default LiveProgress;

// The panel intentionally reports observed milestones rather than inventing a percentage.
// Agent plans can branch or revisit tools, so a numeric estimate would be misleading.
