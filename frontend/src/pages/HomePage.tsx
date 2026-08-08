import { GithubButton } from "@/components/custom/GithubButton"
import { ThemeToggle } from "@/components/custom/ThemeToggle"

function HomePage() {
  return (
    <div className="relative">
      <div className="absolute top-4 right-4">
        <ThemeToggle />
      </div>
      <GithubButton />
    </div>
  )
}

export default HomePage
