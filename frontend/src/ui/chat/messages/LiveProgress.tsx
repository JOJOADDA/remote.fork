import { useEffect, useMemo, useState } from "preact/hooks";
import type { ChatStatus } from "../../../models/chat";
import type { ChatMessageBlock } from "../../../models/chatMessage";

type StageState = "done" | "active" | "queued";

function Stage({
  state,
  label,
  detail,
}: {
  state: StageState;
  label: string;
  detail?: string;
}) {
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
      {detail && <span class="ml-auto max-w-[52%] truncate text-[11px] text-ink-500">{detail}</span>}
    </div>
  );
}

function toolLabel(name: string): string {
  const normalized = name.toLowerCase();
  if (normalized.includes("read") || normalized.includes("cat")) return "Reading files";
  if (normalized.includes("write") || normalized.includes("edit")) return "Editing files";
  if (normalized.includes("bash") || normalized.includes("shell") || normalized.includes("exec")) {
    return "Running a command";
  }
  if (normalized.includes("search") || normalized.includes("grep")) return "Searching the workspace";
  if (normalized.includes("browser") || normalized.includes("chrome")) return "Using the browser";
  return `Using ${name}`;
}

export function LiveProgress({
  blocks,
  status,
}: {
  blocks: ChatMessageBlock[];
  status: ChatStatus;
}) {
  const [now, setNow] = useState(() => Date.now());
  const assistant = [...blocks].reverse().find((block) => block.type === "assistant");
  const parts = assistant?.type === "assistant" ? assistant.parts : [];
  const tools = parts.filter((part) => part.kind === "tool");
  const activeTool = tools.find((part) => part.kind === "tool" && part.status === "running");
  const hasThinking = parts.some((part) => part.kind === "thinking");
  const hasText = parts.some((part) => part.kind === "text");
  const hasTools = tools.length > 0;
  const isActive = status === "streaming";
  const lastActivityAt = Math.max(
    ...blocks.map((block) => block.t),
    ...parts.map(() => 0),
    0
  );
  const idleSeconds = lastActivityAt > 0 ? Math.max(0, Math.floor((now - lastActivityAt) / 1000)) : 0;
  const isStalled = isActive && idleSeconds >= 15;

  useEffect(() => {
    if (!isActive) return;
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [isActive]);

  const currentActivity = useMemo(() => {
    if (activeTool?.kind === "tool") return toolLabel(activeTool.name);
    if (hasThinking) return "Reasoning about the next step";
    if (hasText) return "Preparing the response";
    return "Starting the task";
  }, [activeTool, hasThinking, hasText]);

  const recentTools = tools
    .filter((part) => part.kind === "tool")
    .slice(-3)
    .reverse();

  return (
    <section
      class="mx-auto w-full max-w-3xl overflow-hidden rounded-xl border border-accent-blue/25 bg-[#151a22] shadow-sm"
      role="status"
      aria-live="polite"
      aria-label="Live agent progress"
    >
      <div class="h-1 w-full overflow-hidden bg-white/[0.06]">
        {isActive && <div class="h-full w-1/3 animate-[progress-slide_1.5s_ease-in-out_infinite] rounded-full bg-gradient-to-r from-transparent via-accent-blue to-transparent" />}
      </div>
      <div class="px-3 py-3 sm:px-4">
        <div class="flex items-start justify-between gap-3">
          <div class="flex min-w-0 items-start gap-2">
            <span class={`mt-1.5 h-2 w-2 flex-none rounded-full ${isActive ? "bg-accent-blue animate-pulse" : "bg-accent-green"}`} aria-hidden="true" />
            <div class="min-w-0">
              <div class="truncate text-[13px] font-semibold text-ink-50">
                {isActive ? currentActivity : "Task complete"}
              </div>
              <div class="mt-0.5 text-[11px] text-ink-400">
                {isActive
                  ? isStalled
                    ? `Still working · no new event for ${idleSeconds}s`
                    : "Live activity · waiting for the final result"
                  : "All observed steps finished"}
              </div>
            </div>
          </div>
          <span class="shrink-0 rounded-full border border-accent-blue/20 bg-accent-blue/[0.08] px-2 py-0.5 text-[10px] text-accent-blue">
            {isActive ? "RUNNING" : "DONE"}
          </span>
        </div>

        <div class="mt-3 h-1 rounded-full bg-white/[0.08]" aria-hidden="true">
          <div class={`h-full rounded-full bg-accent-blue transition-all ${isActive ? "w-2/3 animate-pulse" : "w-full bg-accent-green"}`} />
        </div>

        <div class="mt-3 grid grid-cols-1 gap-1.5 sm:grid-cols-2">
          <Stage state={hasThinking || hasText || hasTools ? "done" : isActive ? "active" : "queued"} label="Understand request" />
          <Stage state={hasThinking ? "done" : hasTools || hasText ? "done" : isActive ? "active" : "queued"} label="Reason and plan" />
          <Stage
            state={activeTool ? "active" : hasTools ? "done" : isActive ? "active" : "queued"}
            label="Use tools"
            detail={activeTool?.kind === "tool" ? activeTool.name : hasTools ? `${tools.length} completed` : undefined}
          />
          <Stage state={hasText && !activeTool ? (isActive ? "active" : "done") : isActive ? "active" : "queued"} label="Prepare response" />
        </div>

        {recentTools.length > 0 && (
          <div class="mt-3 border-t border-white/[0.07] pt-2">
            <div class="mb-1 text-[10px] uppercase tracking-wide text-ink-500">Recent activity · {tools.length} step{tools.length === 1 ? "" : "s"}</div>
            <div class="space-y-1">
              {recentTools.map((tool) => (
                <div key={tool.id} class="flex min-w-0 items-center gap-2 text-[11px] text-ink-300">
                  <span class={`h-1.5 w-1.5 flex-none rounded-full ${tool.status === "running" ? "bg-accent-blue animate-pulse" : tool.isError ? "bg-accent-red" : "bg-accent-green"}`} />
                  <span class="truncate">{tool.status === "running" ? toolLabel(tool.name) : tool.name}</span>
                  {tool.status === "done" && <span class="ml-auto shrink-0 text-[10px] text-ink-500">done</span>}
                </div>
              ))}
            </div>
          </div>
        )}

        {isStalled && (
          <div class="mt-3 rounded-md border border-accent-yellow/20 bg-accent-yellow/[0.06] px-2.5 py-2 text-[11px] text-accent-yellow">
            The agent is still running. You can wait for the next tool event or use Cancel if the task is no longer needed.
          </div>
        )}
      </div>
    </section>
  );
}

export default LiveProgress;
