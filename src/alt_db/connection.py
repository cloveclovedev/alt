"""Neon SQL-over-HTTP client."""

import json
import os
import re
import urllib.request
from dataclasses import dataclass

from dotenv import load_dotenv


@dataclass
class QueryResult:
    """Result of a SQL query."""

    rows: list[tuple]
    row_count: int


class NeonHTTP:
    """Neon SQL-over-HTTP client."""

    def __init__(self, host: str, database: str, user: str, password: str):
        self.host = host
        self.database = database
        self.user = user
        self.password = password
        self._url = self._build_url(host)
        self._connection_string = (
            f"postgresql://{user}:{password}@{host}/{database}?sslmode=require"
        )

    @classmethod
    def from_env(cls) -> "NeonHTTP":
        """Create client from NEON_* environment variables."""
        load_dotenv()
        missing = []
        host = os.environ.get("NEON_HOST", "")
        database = os.environ.get("NEON_DATABASE", "")
        user = os.environ.get("NEON_USER", "")
        password = os.environ.get("NEON_PASSWORD", "")
        for name, val in [
            ("NEON_HOST", host),
            ("NEON_DATABASE", database),
            ("NEON_USER", user),
            ("NEON_PASSWORD", password),
        ]:
            if not val:
                missing.append(name)
        if missing:
            raise RuntimeError(
                f"Missing environment variables: {', '.join(missing)}"
            )
        return cls(host, database, user, password)

    @staticmethod
    def _build_url(host: str) -> str:
        """Convert Neon host to HTTP API URL.

        ep-xxx.us-east-2.aws.neon.tech -> https://api.us-east-2.aws.neon.tech/sql
        """
        api_host = re.sub(r"^[^.]+\.", "api.", host)
        return f"https://{api_host}/sql"

    def execute(self, sql: str, params: list | None = None) -> QueryResult:
        """Execute a SQL query via HTTP POST.

        Uses $1, $2, ... for parameter placeholders.
        Returns QueryResult with rows as list[tuple] and row_count.
        """
        body = json.dumps({"query": sql, "params": params or []}).encode()
        req = urllib.request.Request(
            self._url,
            data=body,
            headers={
                "Content-Type": "application/json",
                "Neon-Connection-String": self._connection_string,
                "Neon-Array-Mode": "true",
            },
            method="POST",
        )
        try:
            with urllib.request.urlopen(req) as resp:
                data = json.loads(resp.read())
        except urllib.error.HTTPError as e:
            error_body = e.read().decode()
            try:
                error_data = json.loads(error_body)
                msg = error_data.get("message", error_body)
            except json.JSONDecodeError:
                msg = error_body
            raise RuntimeError(f"Neon query error: {msg}") from e
        except urllib.error.URLError as e:
            raise RuntimeError(f"Neon connection error: {e.reason}") from e

        rows = [tuple(row) for row in data.get("rows", [])]
        row_count = data.get("rowCount", 0)
        return QueryResult(rows=rows, row_count=row_count)
