"""Transport-neutral Phase 0 export contract shell."""

from uuid import uuid4


async def submit(request: dict) -> dict:
    if request.get("version") != "v1" or "payload" not in request:
        raise ValueError("invalid v1 export request")
    return {"version": "v1", "job_ref": f"job-{uuid4().hex}"}


def main() -> None:
    print("plaiflow-export: ok")
