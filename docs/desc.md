# Slate

**Slate** is an open-source, developer-first Platform-as-a-Service (PaaS) designed to orchestrate the entire lifecycle of static and frontend application deployments—from Git webhook to global edge distribution.

Unlike standard web hosts, Slate acts as an ephemeral container orchestrator optimized for single-tenant build isolation, strict host resource governance, and structured artifact distribution. It bridges the gap between high-level developer experience (DX) and low-level Linux/Docker primitives, offering an automated build pipeline that safely compiles untrusted code, optimizes output assets, and maps them to dynamic routing tables.



## Advanced Feature Set & Recruiter-Facing Tech Stack

To stand out to platform and infrastructure teams, the implementation should focus on **low-level systems programming**, **security boundaries**, and **state-machine predictability**.

### Deep Architecture & System Features

* **Dynamic Logs Streaming (SSE/WebSockets):** Multiplex `stdout`/`stderr` from the running build container down to the client UI/CLI in real-time, handling backpressure if the build generates high log volume.
* **Build Interruption & Cleanup Handlers:** Graceful cancellation pipelines using Go contexts or Python signals to instantly SIGKILL containers and reclaim disk/memory if a user cancels a build.
* **Dynamic Reverse Proxy Routing:** A custom routing layer that reads a fast key-value store (like Redis) to dynamically route `*.slate.dev` subdomains to their specific static asset directories in real-time without reloading the proxy config.
* **Content-Addressable Asset Deduplication:** Hashing build assets before uploading to storage to ensure identical files across deployments are stored only once, drastically reducing storage costs.

### Tech Stack Selection & Strategic Alignment

| Component | Technology Options | Why This Impresses Recruiters |
| --- | --- | --- |
| **Orchestration / Logic** | **Go** (Golang) *or* **Python** (Asyncio) | **Go** demonstrates cloud-native alignment (Docker/K8s are written in Go), strict typing, and efficient goroutine concurrency. **Python** showcases mastery of asynchronous execution loops and fast prototyping. |
| **Build Isolation** | **Docker Engine API** (via SDK) | Demonstrates programmatic control over container runtimes, manipulating cgroups, namespaces, and bind mounts directly rather than shelling out to CLI commands. |
| **Pipeline Ingestion** | **Redis Streams** or **RabbitMQ** | Proves understanding of event-driven architectures, consumer groups, dead-letter queues (DLQ), and guarantee levels (at-least-once delivery). |
| **Dynamic Routing** | **Traefik** (with Redis provider) *or* a custom **Go Reverse Proxy** | Shows you know how to build low-latency data planes that don't rely on static config files. |
| **Storage Backend** | **MinIO** (Local S3-compatible API) | Shows architectural parity with production cloud environments (AWS S3) while maintaining a completely self-contained local development environment. |



## The End-to-End Flow

1. **Ingestion:** User pushes code $\rightarrow$ GitHub Webhook hits Slate API $\rightarrow$ API validates the payload and pushes a `BUILD_REQUESTED` event into **Redis Streams**.
2. **Scheduling:** The **Worker Pool** (listening to the stream via consumer groups) claims the job, changing the deployment state to `BUILDING`.
3. **Isolation:** Worker calls the **Docker Engine API** to create a container. It injects strict environment configurations, overrides the entrypoint to run the build script, and sets hard CPU/Memory limits.
4. **Execution & Streaming:** The container boots, clones the code, and runs the build command (e.g., `npm run build`). Worker streams logs out of the container directly to Redis/WebSockets.
5. **Artifact Extraction:** Once compilation succeeds, the worker copies the resulting distribution folder (e.g., `/dist` or `/.next`) via tar-stream, compresses it, and ships it to **MinIO** mapped by a unique deployment hash.
6. **Routing Update:** Worker tears down the container, writes a static distribution manifest, and updates **Redis** keys mapping `deployment-id.slate.dev` to the new asset paths.



##  V1 Blueprint: What You Must Build

To achieve a functional Vercel clone, your system needs to be divided into four decoupled subsystems:

### A. The Control Plane (API Server)

* **REST Endpoints:** Manage projects, trigger manual deployments, and receive webhooks.
* **State Machine:** Track deployment statuses flawlessly: `QUEUED` $\rightarrow$ `BUILDING` $\rightarrow$ `READY` or `FAILED`.
* **Log Broker:** An endpoint that reads logs from a fast buffer and streams them to the user.

### B. The Data Plane / Queue

* **Event Broker:** A Redis Streams architecture configuration handling task visibility timeouts (ensuring if a worker dies, another worker reclaims the build).

### C. The Execution Plane (The Worker Node)

* **Docker Controller:** Code that translates user configuration into raw container configurations.
* **Security Profiles:** Implementation of restricted Linux capabilities (`--cap-drop=ALL`) so the build script cannot exploit the host kernel.
* **Resource Bounds:** Programmatic definition of host constraints:
* `Memory`: Max 512MB
* `NanoCPUs`: Max 1.0 (equivalent to 1 CPU core)
* `Disk`: No-host mount modifications allowed.



### D. The Routing Plane (Edge Proxy)

* A lightweight proxy layer that intercepts incoming HTTP requests, extracts the Host header (e.g., `app-v1.slate.dev`), looks up the file path in Redis, and streams the static file directly from the storage provider or local cache to the client.



## Implementation Roadmap ($V_1$)

```
Phase 1: Core Orchestration (The Engine)
└── Build the Go/Python worker that talks to the Docker API.
└── Hardcode a local directory path, pull a repository, and successfully run a build inside an isolated container.
└── Write the logic to extract the compressed files from the container into local storage.

Phase 2: Event-Driven Pipeline & State
└── Wrap the engine in a Redis Streams pipeline.
└── Build the API server to push tasks to the stream.
└── Implement robust error handling: what happens if a build times out? (Force-kill container, update state to FAILED).

Phase 3: The Edge Router & Manifests
└── Create the routing table layout in Redis.
└── Implement the reverse proxy layer to serve the built files correctly based on the URL domain.
└── Achieve absolute end-to-end flow: Git commit -> automated build -> working URL.

Phase 4: Production Hardening (Recruiter Bait)
└── Apply strict Cgroup resource limits to the Docker API configuration.
└── Implement streaming logs over WebSockets/SSE.
└── Add a trade-off analysis and benchmarking suite comparing your engine's memory footprint under load.

```
