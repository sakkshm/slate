package gateway

import (
	"net/http"
)

const notFoundPage = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta name="color-scheme" content="dark">
<title>404 · Not found</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  html, body { height: 100%; }
  body {
    display: flex;
    align-items: center;
    justify-content: center;
    background: #0f1110;
    color: #f5f5f4;
    font-family: ui-sans-serif, system-ui, -apple-system, "Segoe UI", Roboto, sans-serif;
    -webkit-font-smoothing: antialiased;
    padding: 24px;
  }
  .wrap { text-align: center; max-width: 480px; }
  .mark { display: inline-flex; align-items: center; gap: 8px; font-size: 15px; font-weight: 600; letter-spacing: -0.01em; margin-bottom: 40px; }
  .mark svg { width: 18px; height: 18px; }
  .code { font-size: 96px; font-weight: 700; line-height: 1; letter-spacing: -0.03em; }
  .code span { color: #565b5a; }
  h1 { font-size: 20px; font-weight: 600; margin: 20px 0 8px; }
  p { font-size: 14px; line-height: 1.6; color: #a3a8a7; }
  .actions { margin-top: 32px; display: flex; gap: 12px; justify-content: center; }
  a, button {
    display: inline-flex; align-items: center; justify-content: center;
    height: 38px; padding: 0 16px; border-radius: 8px;
    font-size: 13px; font-weight: 500; text-decoration: none; cursor: pointer;
    border: 1px solid #2a2e2d; background: #171a19; color: #f5f5f4;
    transition: background 0.15s ease, border-color 0.15s ease;
  }
  a:hover, button:hover { background: #202423; border-color: #3a3f3e; }
</style>
</head>
<body>
  <div class="wrap">
    <div class="mark">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M21 8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>
        <path d="m3.3 7 8.7 5 8.7-5"/>
        <path d="M12 22V12"/>
      </svg>
      <span>slate</span>
    </div>
    <div class="code">4<span>0</span>4</div>
    <h1>This site doesn't exist yet</h1>
    <p>The deployment you're looking for hasn't been built, or the address is wrong.</p>
    <div class="actions">
      <a href="https://slate.sakkshm.me">Go to dashboard</a>
    </div>
  </div>
</body>
</html>
`

func writeNotFound(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(notFoundPage))
}
