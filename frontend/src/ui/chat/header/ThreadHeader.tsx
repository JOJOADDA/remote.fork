import type { ChatMeta } from "../../../models/chat";
import { useEffect, useState } from "preact/hooks";
import { providerDisplayLabel } from "../../../config/chat";
import { Menu, MessageSquare } from "../../primitives/icons";

export function ThreadHeader({
  chat,
  streaming,
  onHamburger,
}: {
  chat: ChatMeta;
  streaming: boolean;
  onHamburger: () => void;
}) {
  const [elapsedSeconds, setElapsedSeconds] = useState(0);

  useEffect(() => {
    if (!streaming) {
      setElapsedSeconds(0);
      return;
    }
    const startedAt = Date.now();
    const timer = window.setInterval(() => {
      setElapsedSeconds(Math.floor((Date.now() - startedAt) / 1000));
    }, 1000);
    return () => window.clearInterval(timer);
  }, [streaming]);

  const liveLabel = streaming
    ? `Working · live · ${elapsedSeconds}s`
    : "Ready";

  return (
    <header class="codex-header top-chrome z-20 flex flex-none items-center border-b border-white/10 bg-[#101318] px-3 py-2 md:bg-[#101318]/95 md:backdrop-blur">
      <div class="codex-thread-heading flex min-w-0 flex-1 items-center gap-2 min-h-9">
        <button
          type="button"
          onClick={onHamburger}
          class="md:hidden h-9 w-9 rounded-md text-ink-100 hover:bg-white/[0.08] grid place-items-center flex-none"
          aria-label="Open chats"
          title="Chats"
        >
          <Menu class="w-5 h-5" />
        </button>

        <div class="hidden sm:grid h-8 w-8 rounded-md bg-white/[0.05] border border-white/10 text-ink-300 place-items-center flex-none">
          <MessageSquare class="w-3.5 h-3.5" />
        </div>

        <div class="flex-1 min-w-0">
          <div class="flex items-center gap-2 min-w-0">
            <h1 class="truncate text-[14px] font-semibold text-ink-50">
              {chat.title || "Untitled chat"}
            </h1>
            <span
              class={`h-1.5 w-1.5 rounded-full flex-none ${streaming ? "bg-accent-green animate-pulse" : "bg-ink-400"}`}
              title={streaming ? "Streaming" : "Ready"}
            />
          </div>
          <div class="text-[11px] leading-4 text-ink-400 truncate" aria-live="polite">
            {providerDisplayLabel(chat.provider)} · {liveLabel}
          </div>
        </div>
      </div>

    </header>
  );
}
