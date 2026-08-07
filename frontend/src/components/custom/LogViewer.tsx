import { useEffect, useMemo, useRef, useState } from "react"
import { Copy, Search, Trash2, MousePointerClick } from "lucide-react"
import { toast } from "sonner"

import { subscribeSSE } from "@/shared/api"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { cn } from "@/lib/utils"

const MAX_LINES = 5000

export function LogViewer({
  endpoint,
  onDone,
}: {
  endpoint: string
  onDone?: () => void
}) {
  const [lines, setLines] = useState<string[]>([])
  const [autoScroll, setAutoScroll] = useState(true)
  const [query, setQuery] = useState("")
  const containerRef = useRef<HTMLDivElement>(null)
  const onDoneRef = useRef(onDone)

  useEffect(() => {
    onDoneRef.current = onDone
  }, [onDone])

  useEffect(() => {
    let stopped = false
    const unsubscribe = subscribeSSE(endpoint, {
      onLine: (line) => {
        if (stopped) return
        setLines((prev) => {
          if (prev.length >= MAX_LINES) {
            return [...prev.slice(prev.length - MAX_LINES + 1), line]
          }
          return [...prev, line]
        })
      },
      onDone: () => {
        onDoneRef.current?.()
      },
    })
    return () => {
      stopped = true
      unsubscribe()
    }
  }, [endpoint])

  useEffect(() => {
    const el = containerRef.current
    if (el && autoScroll && !query) {
      el.scrollTop = el.scrollHeight
    }
  }, [lines, autoScroll, query])

  const matches = useMemo(() => {
    if (!query) return null
    return lines
      .map((line, index) => ({ line, index }))
      .filter(({ line }) => line.toLowerCase().includes(query.toLowerCase()))
  }, [lines, query])

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(lines.join("\n"))
      toast.success("Logs copied to clipboard")
    } catch {
      toast.error("Failed to copy logs")
    }
  }

  return (
    <div className="flex flex-col overflow-hidden rounded-xl bg-zinc-950 ring-1 ring-foreground/10">
      <div className="flex items-center gap-2 border-b border-zinc-800 p-2">
        <div className="relative flex-1">
          <Search className="absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-zinc-500" />
          <Input
            className="h-7 border-zinc-800 bg-zinc-900 pl-7 text-xs text-zinc-200 placeholder:text-zinc-600 focus-visible:border-zinc-700"
            placeholder="Filter logs…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </div>
        <Button
          size="sm"
          variant="ghost"
          onClick={() => setAutoScroll((v) => !v)}
          className={cn(
            "h-7 text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100",
            autoScroll && "bg-zinc-800 text-zinc-100"
          )}
        >
          <MousePointerClick />
          Auto-scroll
        </Button>
        <Button
          size="sm"
          variant="ghost"
          onClick={handleCopy}
          className="h-7 text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100"
        >
          <Copy />
          Copy
        </Button>
        <Button
          size="sm"
          variant="ghost"
          onClick={() => setLines([])}
          className="h-7 text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100"
        >
          <Trash2 />
          Clear
        </Button>
      </div>
      <div
        ref={containerRef}
        className="h-[52vh] overflow-y-auto bg-zinc-950 p-3 font-mono text-xs leading-relaxed text-zinc-300"
      >
        {matches !== null ? (
          matches.length === 0 ? (
            <p className="text-zinc-600">No matching lines.</p>
          ) : (
            matches.map(({ line, index }) => (
              <LogLine key={index} line={line} index={index} />
            ))
          )
        ) : lines.length === 0 ? (
          <p className="text-zinc-600">
            Waiting for build logs…
          </p>
        ) : (
          lines.map((line, index) => (
            <LogLine key={index} line={line} index={index} />
          ))
        )}
      </div>
    </div>
  )
}

function LogLine({ line, index }: { line: string; index: number }) {
  return (
    <div className="flex gap-3 whitespace-pre-wrap break-all">
      <span className="shrink-0 text-right text-zinc-700 select-none">
        {index + 1}
      </span>
      <span>{line || " "}</span>
    </div>
  )
}
