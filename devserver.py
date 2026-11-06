#!/usr/bin/env python3
"""Serve web/ and rebuild the wasm when source files change.

Used by ./start.sh --dev. The page polls /__dev and reloads after a build finishes.
"""

import http.server
import os
import subprocess
import threading
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parent
WEB = ROOT / "web"
PORT = int(os.environ.get("PORT", "8000"))

generation = 0
gen_lock = threading.Lock()

RELOAD = b"""
<script>
(function () {
  let seen = null;
  async function poll() {
    try {
      const r = await fetch("/__dev", { cache: "no-store" });
      const t = await r.text();
      if (seen !== null && t !== seen) location.reload();
      seen = t;
    } catch (e) {}
    setTimeout(poll, 400);
  }
  poll();
})();
</script>
"""


def interesting(path: Path) -> bool:
    rel = path.relative_to(ROOT)
    if rel.as_posix() == "assets/maps/world.yaml":
        return False  # the map editor writes this; it must not rebuild and reload the page
    parts = rel.parts
    if not parts or parts[0] in {"web", ".git"} or any(p.startswith(".") for p in parts):
        return False
    if parts[0] in {"assets", "_res"}:
        return path.is_file()
    return path.suffix == ".go" or path.name in {"go.mod", "go.sum"}


def snapshot() -> dict[Path, tuple[int, int]]:
    found = {}
    for dirpath, dirnames, filenames in os.walk(ROOT):
        dirnames[:] = [d for d in dirnames if not d.startswith(".") and d not in {"web", ".git"}]
        for name in filenames:
            path = Path(dirpath) / name
            if not interesting(path):
                continue
            try:
                st = path.stat()
            except OSError:
                continue
            found[path] = (st.st_mtime_ns, st.st_size)
    index = WEB / "index.html"
    try:
        st = index.stat()
    except OSError:
        pass
    else:
        found[index] = (st.st_mtime_ns, st.st_size)
    return found


def changed_paths(prev: dict, cur: dict) -> list[Path]:
    keys = set(prev) | set(cur)
    return [p for p in keys if prev.get(p) != cur.get(p)]


def build() -> bool:
    print("rebuilding…", flush=True)
    result = subprocess.run(["./build-web.sh"], cwd=ROOT)
    if result.returncode != 0:
        print("build failed; still serving the last wasm", flush=True)
        return False
    return True


def bump() -> None:
    global generation
    with gen_lock:
        generation += 1


class Handler(http.server.SimpleHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, directory=str(WEB), **kwargs)

    def log_message(self, fmt, *args):
        if self.path.split("?", 1)[0] == "/__dev":
            return
        super().log_message(fmt, *args)

    def end_headers(self):
        path = self.path.split("?", 1)[0]
        if path in {"/game.wasm", "/wasm_exec.js", "/", "/index.html", "/__dev"}:
            self.send_header("Cache-Control", "no-store")
        super().end_headers()

    def do_POST(self):
        path = self.path.split("?", 1)[0]
        if path != "/__map":
            self.send_error(404)
            return
        try:
            n = int(self.headers.get("Content-Length", "0"))
        except ValueError:
            self.send_error(400)
            return
        if n < 0 or n > 2_000_000:
            self.send_error(413)
            return
        body = self.rfile.read(n)
        dest = ROOT / "assets" / "maps" / "world.yaml"
        dest.parent.mkdir(parents=True, exist_ok=True)
        tmp = dest.with_suffix(".yaml.tmp")
        tmp.write_bytes(body)
        tmp.replace(dest)
        self.send_response(204)
        self.end_headers()

    def do_GET(self):
        path = self.path.split("?", 1)[0]
        if path == "/assets/maps/world.yaml":
            self.serve_file(ROOT / "assets" / "maps" / "world.yaml", "application/yaml; charset=utf-8")
            return
        if path == "/__dev":
            with gen_lock:
                body = str(generation).encode()
            self.send_response(200)
            self.send_header("Content-Type", "text/plain; charset=utf-8")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return
        if path in {"/", "/index.html"}:
            self.serve_index()
            return
        super().do_GET()

    def serve_file(self, path: Path, content_type: str):
        try:
            data = path.read_bytes()
        except OSError:
            self.send_error(404)
            return
        self.send_response(200)
        self.send_header("Content-Type", content_type)
        self.send_header("Content-Length", str(len(data)))
        self.send_header("Cache-Control", "no-store")
        self.end_headers()
        self.wfile.write(data)

    def serve_index(self):
        data = (WEB / "index.html").read_bytes()
        if b"</body>" in data:
            data = data.replace(b"</body>", RELOAD + b"</body>", 1)
        else:
            data += RELOAD
        self.send_response(200)
        self.send_header("Content-Type", "text/html; charset=utf-8")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)


def watch() -> None:
    prev = snapshot()
    while True:
        time.sleep(0.4)
        cur = snapshot()
        changed = changed_paths(prev, cur)
        if not changed:
            continue
        time.sleep(0.3)  # let a burst of writes settle
        cur = snapshot()
        changed = changed_paths(prev, cur)
        prev = cur
        needs_build = any(p.name != "index.html" for p in changed)
        if needs_build:
            if build():
                bump()
        else:
            bump()
        prev = snapshot()


def main() -> None:
    watch_sources = os.environ.get("JEWEL_WATCH", "1") != "0"
    if watch_sources:
        if not build():
            raise SystemExit(1)
        threading.Thread(target=watch, daemon=True).start()
    print(f"Open http://localhost:{PORT}", flush=True)
    print("Saving a source, asset, or story file rebuilds the wasm and reloads the page.", flush=True)
    http.server.ThreadingHTTPServer(("127.0.0.1", PORT), Handler).serve_forever()


if __name__ == "__main__":
    main()
