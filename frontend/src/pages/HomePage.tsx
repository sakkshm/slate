import {
  ArrowRight,
  Box,
  Container,
  Database,
  GitBranch,
  Globe,
  KeyRound,
  Rocket,
  ShieldCheck,
  Terminal,
} from "lucide-react"

import { GithubButton } from "@/components/custom/GithubButton"
import { ThemeToggle } from "@/components/custom/ThemeToggle"

const FEATURES = [
  {
    icon: Rocket,
    title: "Deploy on every push",
    description:
      "Connect a GitHub repo once. Every push after that is built and deployed for you. No CI config, nothing to babysit.",
  },
  {
    icon: Container,
    title: "Isolated builds",
    description:
      "Each build runs in its own memory- and CPU-limited container with a locked-down network.",
  },
  {
    icon: Terminal,
    title: "Live build logs",
    description:
      "Watch logs stream into your browser in real time. If a build breaks, you'll see exactly why, right as it happens.",
  },
  {
    icon: KeyRound,
    title: "Environment variables",
    description:
      "Keep secrets per project, right from the dashboard. Values stay encrypted at rest.",
  },
  {
    icon: Globe,
    title: "Custom subdomains",
    description:
      "Every deployment is live instantly at your slug. HTTPS included.",
  },
  {
    icon: ShieldCheck,
    title: "Content-addressed gateway",
    description:
      "Artifacts are stored by SHA-256 hash and served by an in-process gateway. Immutable, reproducible, and fast.",
  },
]

const STEPS = [
  {
    number: "01",
    icon: GitBranch,
    title: "Push",
    description:
      "Git push to your repository, and a GitHub webhook wakes Slate up.",
  },
  {
    number: "02",
    icon: Container,
    title: "Build",
    description:
      "A worker builds it in an isolated, resource-limited container while logs stream live.",
  },
  {
    number: "03",
    icon: Database,
    title: "Store",
    description:
      "The output is stored as a content-addressed artifact, keyed by its SHA-256 hash.",
  },
  {
    number: "04",
    icon: Globe,
    title: "Serve",
    description:
      "The gateway puts it live instantly at <slug>.slate.sakkshm.me.",
  },
]

function Nav() {
  return (
    <nav className="sticky top-0 z-20 border-b border-border/60 bg-background/70 backdrop-blur">
      <div className="mx-auto flex h-14 w-full max-w-6xl items-center justify-between px-4">
        <a href="#top" className="flex items-center gap-2">
          <Box className="size-5" />
          <span className="font-heading text-base font-semibold tracking-tight">
            slate
          </span>
        </a>
        <div className="flex items-center gap-4">
          <a
            href="#features"
            className="hidden text-sm text-muted-foreground transition-colors hover:text-foreground sm:block"
          >
            Features
          </a>
          <a
            href="#how-it-works"
            className="hidden text-sm text-muted-foreground transition-colors hover:text-foreground sm:block"
          >
            How it works
          </a>
          <ThemeToggle />
          <GithubButton label="Sign in" />
        </div>
      </div>
    </nav>
  )
}

function Hero() {
  return (
    <section className="mx-auto flex min-h-[calc(100svh-3.5rem)] w-full max-w-4xl flex-col items-center justify-center px-4 py-24 text-center">
      <h1 className="font-heading text-4xl font-semibold tracking-tight sm:text-6xl">
        Push to GitHub.
        <br />
        We ship it.
      </h1>
      <p className="mx-auto mt-6 max-w-2xl text-base text-muted-foreground sm:text-lg">
        Slate is a platform that builds your websites in isolated containers, stores the output, and publishes them live on
        with every push.
      </p>
      <div className="mt-10 flex flex-col items-center justify-center gap-3 sm:flex-row">
        <GithubButton label="Sign in with GitHub" />
        <a
          href="#how-it-works"
          className="inline-flex h-9 items-center gap-1.5 rounded-lg border border-border bg-background px-3 text-sm font-medium text-foreground transition-colors hover:bg-muted"
        >
          How it works
          <ArrowRight data-icon="inline-end" />
        </a>
      </div>
    </section>
  )
}

function Features() {
  return (
    <section id="features" className="mx-auto w-full max-w-6xl px-4 py-20">
      <h2 className="font-heading text-3xl font-semibold tracking-tight">
        Everything you need to ship
      </h2>
      <p className="mt-3 max-w-2xl text-muted-foreground">
        Build, logs, secrets, and serving in one place. No infrastructure to
        manage.
      </p>
      <div className="mt-10 grid gap-px overflow-hidden rounded-2xl border border-border bg-border sm:grid-cols-2 lg:grid-cols-3">
        {FEATURES.map((feature) => (
          <div
            key={feature.title}
            className="bg-background p-6 transition-colors hover:bg-muted/40"
          >
            <feature.icon className="size-5 text-foreground" />
            <h3 className="mt-4 font-heading text-base font-semibold tracking-tight">
              {feature.title}
            </h3>
            <p className="mt-1.5 text-sm leading-relaxed text-muted-foreground">
              {feature.description}
            </p>
          </div>
        ))}
      </div>
    </section>
  )
}

function HowItWorks() {
  return (
    <section
      id="how-it-works"
      className="mx-auto w-full max-w-6xl px-4 py-20"
    >
      <h2 className="font-heading text-3xl font-semibold tracking-tight">
        How it works
      </h2>
      <p className="mt-3 max-w-2xl text-muted-foreground">
        From git push to a live site in one pipeline, built on Redis streams,
        Docker, and MinIO.
      </p>
      <div className="mt-10 grid gap-px overflow-hidden rounded-2xl border border-border bg-border sm:grid-cols-2 lg:grid-cols-4">
        {STEPS.map((step) => (
          <div
            key={step.number}
            className="group bg-background p-6 transition-colors hover:bg-muted/40"
          >
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium tabular-nums text-muted-foreground">
                {step.number}
              </span>
              <step.icon className="size-4 text-muted-foreground transition-colors group-hover:text-foreground" />
            </div>
            <h3 className="mt-4 font-heading text-base font-semibold tracking-tight">
              {step.title}
            </h3>
            <p className="mt-1.5 text-sm leading-relaxed text-muted-foreground">
              {step.description}
            </p>
          </div>
        ))}
      </div>
    </section>
  )
}

function CTA() {
  return (
    <section className="mx-auto w-full max-w-6xl px-4 py-20">
      <div className="flex flex-col items-center gap-6 rounded-2xl border border-border bg-background px-6 py-16 text-center">
        <h2 className="font-heading text-3xl font-semibold tracking-tight">
          Ready to ship?
        </h2>
        <p className="max-w-xl text-muted-foreground">
          Connect your GitHub account, pick a repo, and your first deployment is
          live in minutes.
        </p>
        <GithubButton label="Sign in with GitHub" />
      </div>
    </section>
  )
}

function Footer() {
  return (
    <footer className="border-t border-border/60">
      <div className="mx-auto flex w-full max-w-6xl flex-col items-center justify-between gap-3 px-4 py-8 sm:flex-row">
        <div className="flex items-center gap-2">
          <Box className="size-4" />
          <span className="font-heading text-sm font-semibold tracking-tight">
            slate
          </span>
        </div>
        <p className="text-xs text-muted-foreground">
          Build and deploy static sites on every push.
        </p>
      </div>
    </footer>
  )
}

export default function HomePage() {
  return (
    <div id="top" className="relative min-h-svh">
      <div className="bg-grid pointer-events-none fixed inset-0 z-0" />
      <div className="relative z-10">
        <Nav />
        <Hero />
        <Features />
        <HowItWorks />
        <CTA />
        <Footer />
      </div>
    </div>
  )
}
