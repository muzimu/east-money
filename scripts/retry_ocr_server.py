#!/usr/bin/env python3
"""Minimal OCR retry test server.

Behavior:
- POST /ocr #1 and #2: return fixed wrong value "1234"
- POST /ocr #3 and later: proxy request body to the upstream OCR endpoint
- GET /status: inspect counter
- POST /reset or GET /reset: reset counter to 0
"""

from __future__ import annotations

import argparse
import json
import threading
import urllib.error
import urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer


class RetryOCRHandler(BaseHTTPRequestHandler):
    counter = 0
    lock = threading.Lock()
    upstream = ""

    def log_message(self, fmt: str, *args) -> None:
        print("[%s] %s" % (self.log_date_time_string(), fmt % args), flush=True)

    def _write_json(self, status: int, payload: dict) -> None:
        body = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self) -> None:
        if self.path == "/status":
            self._write_json(200, {"ok": True, "count": self.counter, "upstream": self.upstream})
            return
        if self.path == "/reset":
            with self.lock:
                self.__class__.counter = 0
            self._write_json(200, {"ok": True, "count": 0})
            return
        self._write_json(404, {"ok": False, "message": "not found"})

    def do_POST(self) -> None:
        if self.path == "/reset":
            with self.lock:
                self.__class__.counter = 0
            self._write_json(200, {"ok": True, "count": 0})
            return

        if self.path != "/ocr":
            self._write_json(404, {"ok": False, "message": "not found"})
            return

        content_length = int(self.headers.get("Content-Length", "0"))
        body = self.rfile.read(content_length)

        with self.lock:
            self.__class__.counter += 1
            current = self.__class__.counter

        if current <= 2:
            self._write_json(200, {"code": 200, "data": "1234", "message": "success", "attempt": current})
            return

        req = urllib.request.Request(
            self.upstream,
            data=body,
            method="POST",
            headers={
                "Content-Type": self.headers.get("Content-Type", "application/octet-stream"),
                "Content-Length": str(len(body)),
                "User-Agent": "east-money-retry-ocr-test/1.0",
            },
        )
        try:
            with urllib.request.urlopen(req, timeout=20) as resp:
                upstream_body = resp.read()
                self.send_response(resp.status)
                self.send_header("Content-Type", resp.headers.get("Content-Type", "application/json; charset=utf-8"))
                self.send_header("Content-Length", str(len(upstream_body)))
                self.end_headers()
                self.wfile.write(upstream_body)
        except urllib.error.HTTPError as exc:
            err_body = exc.read()
            self.send_response(exc.code)
            self.send_header("Content-Type", exc.headers.get("Content-Type", "application/json; charset=utf-8"))
            self.send_header("Content-Length", str(len(err_body)))
            self.end_headers()
            self.wfile.write(err_body)
        except Exception as exc:  # noqa: BLE001 - tiny diagnostic service
            self._write_json(502, {"code": 502, "data": "", "message": f"upstream OCR failed: {exc}"})


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=18080)
    parser.add_argument("--upstream", required=True)
    args = parser.parse_args()

    RetryOCRHandler.upstream = args.upstream
    server = ThreadingHTTPServer((args.host, args.port), RetryOCRHandler)
    print(f"listening on http://{args.host}:{args.port}/ocr", flush=True)
    print(f"upstream: {args.upstream}", flush=True)
    server.serve_forever()


if __name__ == "__main__":
    main()
