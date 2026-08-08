import { useState } from "react"
import { Moon, Sun } from "lucide-react"
import { useTheme } from "next-themes"

import { Button } from "@/components/ui/button"

export function ThemeToggle() {
  const { setTheme } = useTheme()
  const [isDark, setIsDark] = useState(() => {
    if (typeof document === "undefined") return false
    return document.documentElement.classList.contains("dark")
  })

  const handleToggle = () => {
    const next = !isDark
    setIsDark(next)
    setTheme(next ? "dark" : "light")
  }

  return (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      aria-label={isDark ? "Switch to light theme" : "Switch to dark theme"}
      onClick={handleToggle}
      className="text-muted-foreground hover:text-foreground"
    >
      {isDark ? <Sun /> : <Moon />}
    </Button>
  )
}
