"""Minimal HTTP server that logs POST bodies to stdout."""

import json
from http.server import HTTPServer, BaseHTTPRequestHandler


class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        length = int(self.headers.get("Content-Length", 0))
        body = self.rfile.read(length)
        try:
            data = json.loads(body)
            print(json.dumps(data, indent=2), flush=True)
        except json.JSONDecodeError:
            print(body.decode(), flush=True)
        self.send_response(200)
        self.end_headers()

    def log_message(self, fmt, *args):
        # Suppress default request logging
        pass


if __name__ == "__main__":
    server = HTTPServer(("0.0.0.0", 8888), Handler)
    print("log-sink listening on :8888", flush=True)
    server.serve_forever()
