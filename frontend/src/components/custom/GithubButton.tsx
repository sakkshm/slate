import { IconBrandGithub } from "@tabler/icons-react"
import { Button } from "@/components/ui/button"

export function GithubButton() {
  return (
    <div className="flex gap-2">
      <Button>
        <IconBrandGithub data-icon="inline-start" /> Connect Github
      </Button>
    </div>
  )
}