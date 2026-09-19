#!/usr/bin/env python3
"""One door to the GPT (codex) and Gemini (agy) CLIs for prose and its review.

Anthropic models do the engineering here; text that reaches a reader and every verdict
about such text goes through a second model family. This script is the only way skills
and agents reach those CLIs, so the isolation, the model allowlist and the mechanical
output check live in one place.

Exit codes: 0 ok, 2 no engine available (stop and report, never write the text yourself),
3 the engine failed, 4 the output failed validation, 5 bad usage.
"""
import argparse
import json
import os
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

GPT_MODEL_ENV = "TEXT_MODEL_GPT"
GEMINI_MODEL_ENV = "TEXT_MODEL_GEMINI"
# agy identifiers carry the effort level; "pro" has no medium tier, so medium maps to high.
GEMINI_MODELS = {"low": "gemini-3.1-pro-low", "medium": "gemini-3.1-pro-high", "high": "gemini-3.1-pro-high"}

EXIT_NO_ENGINE, EXIT_ENGINE_FAILED, EXIT_INVALID_OUTPUT, EXIT_USAGE = 2, 3, 4, 5


def detect():
    return {"gpt": shutil.which("codex") is not None, "gemini": shutil.which("agy") is not None}


def build_prompt(prompt, inputs, schema):
    parts = [prompt.strip(), ""]
    for path in inputs:
        parts.append('<input name="%s">' % path.name)
        parts.append(path.read_text(encoding="utf-8"))
        parts.append("</input>")
        parts.append("")
    if schema is not None:
        parts.append("Answer with one JSON document that validates against this schema and nothing else:")
        parts.append(json.dumps(schema))
    return "\n".join(parts)


def run(cmd, stdin_text, timeout):
    try:
        return subprocess.run(cmd, input=stdin_text, capture_output=True, text=True, timeout=timeout, check=False)
    except subprocess.TimeoutExpired:
        return None


def run_gpt(prompt, schema_path, effort, timeout):
    model = os.environ.get(GPT_MODEL_ENV)
    with tempfile.TemporaryDirectory(prefix="text-model-") as workdir:
        out_path = Path(workdir) / "last-message.txt"
        cmd = [
            "codex", "exec", "--sandbox", "read-only", "--ephemeral", "--skip-git-repo-check",
            "--color", "never", "-c", 'model_reasoning_effort="%s"' % effort,
            "-C", workdir, "-o", str(out_path),
        ]
        if model:
            cmd += ["-m", model]
        if schema_path is not None:
            cmd += ["--output-schema", str(schema_path)]
        cmd.append("-")
        result = run(cmd, prompt, timeout)
        if result is None:
            return None, "codex: timed out after %ss" % timeout
        if result.returncode != 0 or not out_path.exists():
            return None, "codex: exit %s\n%s" % (result.returncode, result.stderr[-2000:])
        return out_path.read_text(encoding="utf-8"), None


def run_gemini(prompt, schema_path, effort, timeout):
    model = os.environ.get(GEMINI_MODEL_ENV, GEMINI_MODELS[effort])
    if not model.startswith("gemini-"):
        return None, "agy: only gemini-* models pass through this door, got %s" % model
    cmd = ["agy", "--print=" + prompt, "--model", model, "--disable-slash-commands", "--sandbox"]
    if schema_path is not None:
        cmd += ["--output-format", "json", "--json-schema", str(schema_path)]
    else:
        cmd += ["--output-format", "text"]
    result = run(cmd, "", timeout)
    if result is None:
        return None, "agy: timed out after %ss" % timeout
    if result.returncode != 0:
        return None, "agy: exit %s\n%s" % (result.returncode, (result.stderr or result.stdout)[-2000:])
    if schema_path is None:
        return result.stdout, None
    try:
        envelope = json.loads(result.stdout)
    except ValueError:
        return None, "agy: --output-format json did not return JSON:\n%s" % result.stdout[-2000:]
    structured = envelope.get("structured_output") if isinstance(envelope, dict) else None
    if structured is None:
        return None, "agy: envelope carries no structured_output:\n%s" % result.stdout[-2000:]
    return json.dumps(structured, ensure_ascii=False, indent=2), None


def validate(text, schema):
    if not text or not text.strip():
        return "empty output"
    if schema is None:
        return None
    try:
        doc = json.loads(text)
    except ValueError as err:
        return "output is not JSON: %s" % err
    missing = [k for k in schema.get("required", []) if not (isinstance(doc, dict) and k in doc)]
    if missing:
        return "output lacks required keys: %s" % ", ".join(missing)
    return None


def main(argv):
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--detect", action="store_true", help="print which engines are installed and exit")
    parser.add_argument("--prompt", help="the task, as text")
    parser.add_argument("--prompt-file", type=Path, help="the task, read from a file")
    parser.add_argument("--input", type=Path, nargs="*", default=[], help="files appended to the prompt as <input> blocks")
    parser.add_argument("--out", type=Path, help="where the answer is written (both: <out>.gpt and <out>.gemini)")
    parser.add_argument("--schema", type=Path, help="JSON schema the answer must validate against")
    parser.add_argument("--engine", choices=["auto", "gpt", "gemini", "both"], default="auto")
    parser.add_argument("--effort", choices=["low", "medium", "high"], default="medium")
    parser.add_argument("--timeout", type=int, default=900, help="seconds per engine call")
    args = parser.parse_args(argv)

    available = detect()
    if args.detect:
        print(json.dumps(available))
        return 0
    if args.prompt is None and args.prompt_file is None:
        parser.error("--prompt or --prompt-file is required")
    if args.out is None:
        parser.error("--out is required")
    prompt = args.prompt if args.prompt is not None else args.prompt_file.read_text(encoding="utf-8")
    schema = json.loads(args.schema.read_text(encoding="utf-8")) if args.schema else None

    engines = {"gpt": run_gpt, "gemini": run_gemini}
    if args.engine == "auto":
        wanted = [e for e in ("gpt", "gemini") if available[e]][:1]
    elif args.engine == "both":
        wanted = ["gpt", "gemini"]
    else:
        wanted = [args.engine]
    missing = [e for e in wanted if not available[e]]
    if not wanted or missing:
        print("no content engine available (%s). Do not write this text yourself: stop and report."
              % (", ".join(missing) or "codex and agy both absent"), file=sys.stderr)
        return EXIT_NO_ENGINE

    full_prompt = build_prompt(prompt, args.input, schema)
    status = 0
    for engine in wanted:
        text, err = engines[engine](full_prompt, args.schema, args.effort, args.timeout)
        if err is not None:
            print("%s failed: %s" % (engine, err), file=sys.stderr)
            status = EXIT_ENGINE_FAILED
            continue
        problem = validate(text, schema)
        target = args.out if len(wanted) == 1 else args.out.with_name(args.out.name + "." + engine)
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(text, encoding="utf-8")
        if problem is not None:
            print("%s output rejected: %s (kept at %s)" % (engine, problem, target), file=sys.stderr)
            status = status or EXIT_INVALID_OUTPUT
            continue
        print("%s ok -> %s (%d bytes)" % (engine, target, len(text.encode("utf-8"))))
    return status


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
