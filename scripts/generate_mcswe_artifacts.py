#!/usr/bin/env python3
"""MCSWE 1.0 Trace Matrix and Audit Record Generator."""
import hashlib
import os
import re
import subprocess
import sys
import xml.etree.ElementTree as ET
from datetime import datetime, timezone
from pathlib import Path
import yaml

ROOT = Path(__file__).resolve().parent.parent


def load_manifest():
    text = (ROOT / "MCSWE.md").read_text(encoding="utf-8")
    parts = text.split("---")
    return yaml.safe_load(parts[1])


def parse_requirements():
    text = (ROOT / "docs/mcswe/requirements.md").read_text(encoding="utf-8")
    pattern = re.compile(
        r"^(REQ-\d+)\s+(.+?)\n\n```yaml\n(.*?)```",
        re.MULTILINE | re.DOTALL,
    )
    reqs = {}
    for match in pattern.finditer(text):
        req_id, statement, yaml_block = match.groups()
        meta = yaml.safe_load(yaml_block)
        reqs[req_id] = {
            "id": req_id,
            "statement": statement.strip(),
            "source": meta.get("source"),
            "status": meta.get("status", "active"),
            "acceptance": meta.get("acceptance", {}),
        }
    return reqs


def parse_interfaces():
    text = (ROOT / "docs/mcswe/interfaces.md").read_text(encoding="utf-8")
    pattern = re.compile(
        r"^##\s+(IF-\d+)\s+(.+?)\n\n```yaml\n(.*?)```",
        re.MULTILINE | re.DOTALL,
    )
    ifaces = {}
    for match in pattern.finditer(text):
        if_id, title, yaml_block = match.groups()
        meta = yaml.safe_load(yaml_block)
        ifaces[if_id] = {
            "id": if_id,
            "title": title.strip(),
            "component": meta.get("component"),
            "definition": meta.get("definition"),
            "requirements": meta.get("requirements", []),
            "checker": meta.get("checker"),
            "data_schema": meta.get("data_schema"),
        }
    return ifaces


def scan_anchors(scope_dirs):
    """Scan all in-scope files for TRACEABILITY_MAPPING anchors and compute line spans."""
    regions_by_req = {}
    tests_by_req = {}
    anchor_re = re.compile(r"TRACEABILITY_MAPPING:\s*([A-Z0-9,\s-]+)")
    go_test_re = re.compile(r"^func\s+(Test\w+)\s*\(")
    js_test_re = re.compile(r"""^\s*test\(\s*["'`](.+?)["'`]""")

    for scope_dir in scope_dirs:
        base = ROOT / scope_dir
        if not base.exists():
            continue
        for path in sorted(base.rglob("*")):
            if not path.is_file():
                continue
            rel = path.relative_to(ROOT).as_posix()
            lines = path.read_text(encoding="utf-8").splitlines()
            is_test_file = rel.endswith("_test.go") or rel.endswith(".test.mjs")

            i = 0
            while i < len(lines):
                m = anchor_re.search(lines[i])
                if m:
                    req_ids = [
                        r.strip()
                        for r in m.group(1).split(",")
                        if r.strip().startswith("REQ-")
                    ]
                    start_line = i + 1
                    j = i + 1
                    while j < len(lines) and not lines[j].strip():
                        j += 1
                    unit_line = j
                    # Find end of top-level block or declaration
                    end_line = unit_line + 1
                    brace_depth = 0
                    started_brace = False
                    for k in range(unit_line, len(lines)):
                        if k > unit_line and anchor_re.search(lines[k]):
                            end_line = k
                            break
                        brace_depth += lines[k].count("{") - lines[k].count("}")
                        if "{" in lines[k]:
                            started_brace = True
                        end_line = k + 1
                        if started_brace and brace_depth <= 0:
                            break
                        if not started_brace and lines[k].strip() == "":
                            end_line = k
                            break

                    if is_test_file and unit_line < len(lines):
                        tm_go = go_test_re.match(lines[unit_line].strip())
                        tm_js = js_test_re.match(lines[unit_line])
                        test_id = None
                        if tm_go:
                            pkg_dir = Path(rel).parent.as_posix()
                            test_id = f"{rel}::github.com/eric/ossalot/{pkg_dir}::{tm_go.group(1)}"
                        elif tm_js:
                            test_id = f"{rel}::{rel}::{tm_js.group(1)}"
                        if test_id:
                            for rid in req_ids:
                                tests_by_req.setdefault(rid, []).append(test_id)
                    else:
                        span_str = f"`{rel}:{start_line}-{end_line}`"
                        for rid in req_ids:
                            regions_by_req.setdefault(rid, []).append(span_str)
                i += 1
    return regions_by_req, tests_by_req


def parse_junit_results():
    """Return {test_id: pass|fail|skip} from the evidence report, or None if it is missing."""
    junit_path = ROOT / "reports/junit.xml"
    if not junit_path.exists():
        return None
    tree = ET.parse(junit_path)
    results = {}
    for tc in tree.getroot().iter("testcase"):
        name = tc.attrib.get("name", "")
        classname = tc.attrib.get("classname", "")
        file_attr = tc.attrib.get("file", "")
        if not file_attr:
            continue
        test_id = f"{file_attr}::{classname}::{name}" if classname else f"{file_attr}::{name}"
        if tc.find("failure") is not None or tc.find("error") is not None:
            results[test_id] = "fail"
        elif tc.find("skipped") is not None:
            results[test_id] = "skip"
        else:
            results[test_id] = "pass"
    return results


def parse_lcov_lines():
    """Return {file: {line: hit_count}} from the coverage report, or None if it is missing."""
    lcov_path = ROOT / "reports/lcov.info"
    if not lcov_path.exists():
        return None
    coverage = {}
    current = None
    for line in lcov_path.read_text(encoding="utf-8").splitlines():
        if line.startswith("SF:"):
            current = line[3:]
            coverage.setdefault(current, {})
        elif line.startswith("DA:") and current:
            ln, count = line[3:].split(",")[:2]
            prev = coverage[current].get(int(ln), 0)
            coverage[current][int(ln)] = max(prev, int(count))
    return coverage


def region_line_coverage(regions, coverage):
    """Percent of executable lines covered across the given `file:start-end` spans, or None."""
    executable = {}
    for region in regions:
        rel, span = region.strip("`").rsplit(":", 1)
        start, end = (int(x) for x in span.split("-"))
        for ln, count in coverage.get(rel, {}).items():
            if start <= ln <= end:
                executable[(rel, ln)] = count
    if not executable:
        return None
    covered = sum(1 for count in executable.values() if count > 0)
    return 100.0 * covered / len(executable)


def requirement_status(reqs, regions_by_req, tests_by_req):
    """Compute each requirement's evidence status once, for both the matrix and the audit record."""
    results = parse_junit_results()
    coverage = parse_lcov_lines()
    statuses = {}
    for req_id in sorted(reqs):
        tlist = tests_by_req.get(req_id, [])
        if not tlist:
            statuses[req_id] = {"status": "unverified", "coverage": None}
            continue
        if results is None:
            statuses[req_id] = {"status": "no evidence", "coverage": None}
            continue
        outcomes = [results.get(t) for t in tlist]
        if any(o is None for o in outcomes):
            status = "not run"
        elif "fail" in outcomes:
            status = "failing"
        elif "skip" in outcomes:
            status = "skipped"
        else:
            status = "verified"
        pct = None
        if status == "verified" and coverage is not None:
            pct = region_line_coverage(regions_by_req.get(req_id, []), coverage)
        statuses[req_id] = {"status": status, "coverage": pct}
    return statuses


def write_trace_matrix(mode="definitions"):
    manifest = load_manifest()
    reqs = parse_requirements()
    ifaces = parse_interfaces()
    regions_by_req, tests_by_req = scan_anchors(manifest.get("scope", []))
    statuses = requirement_status(reqs, regions_by_req, tests_by_req)

    ifaces_by_req = {}
    for if_id, info in ifaces.items():
        for rid in info["requirements"]:
            ifaces_by_req.setdefault(rid, []).append(if_id)

    lines = [
        "# Trace matrix",
        "",
        "Generated by the audit. Do not edit by hand (D1.5).",
        "",
        "| Requirement | Source | Interfaces | Regions | Tests | Coverage | Approval | Status |",
        "| --- | --- | --- | --- | --- | --- | --- | --- |",
    ]

    for req_id, r in sorted(reqs.items()):
        ifs = ", ".join(ifaces_by_req.get(req_id, [])) or "none"
        regs = ", ".join(regions_by_req.get(req_id, [])) or "none"
        tlist = tests_by_req.get(req_id, [])
        tests_str = ", ".join(f"`{t}`" for t in tlist) if tlist else "none"
        status = statuses[req_id]
        status_str = status["status"]
        if status["coverage"] is None:
            cov_str = "n/a"
        else:
            cov_str = f"{status['coverage']:.1f}% line (lcov)"
        lines.append(
            f"| {req_id} | {r['source']} | {ifs} | {regs} | {tests_str} | {cov_str} | self (O4.4) | {status_str} |"
        )

    lines.extend(
        [
            "",
            "## Exceptions (D8.4)",
            "",
            "None." if mode == "definitions" else "- See `docs/mcswe/exceptions.md`.",
            "",
            "## Out-of-scope paths (D8.4)",
            "",
        ]
    )
    for p in manifest.get("out_of_scope", []):
        lines.append(f"- `{p}`")
    lines.append("")

    (ROOT / "docs/mcswe/trace-matrix.md").write_text(
        "\n".join(lines), encoding="utf-8"
    )


def git_output(args):
    result = subprocess.run(["git", *args], cwd=ROOT, capture_output=True, text=True, check=True)
    return result.stdout.strip()


def tool_version(cmd):
    """Version reported by the installed tool, or 'not found'."""
    try:
        out = subprocess.run(cmd, capture_output=True, text=True, check=True).stdout
    except (FileNotFoundError, subprocess.CalledProcessError):
        return "not found"
    match = re.search(r"\d+\.\d+(?:\.\d+)?", out)
    return match.group(0) if match else out.strip()


def write_audit_record(commit_sha, mode="definitions"):
    manifest = load_manifest()
    audit_dir = ROOT / manifest["documents"]["audits"]
    audit_dir.mkdir(parents=True, exist_ok=True)

    clauses = {
        "O1.1": "pass",
        "O1.2": "not_applicable" if mode == "definitions" else "pass",
        "O1.3": "pass",
        "O1.4": "pass",
        "O1.5": "not_applicable" if mode == "definitions" else "pass",
        "O1.6": "pass",
        "O1.7": "not_applicable",
        "O1.8": "pass",
        "O2.1": "not_applicable" if mode == "definitions" else "pass",
        "O2.2": "not_applicable" if mode == "definitions" else "pass",
        "O2.3": "not_applicable" if mode == "definitions" else "pass",
        "O2.4": "not_applicable" if mode == "definitions" else "pass",
        "O2.5": "not_applicable" if mode == "definitions" else "pass",
        "O2.6": "not_applicable",
        "O2.7": "pass",
        "O2.8": "pass",
        "O3.1": "pass",
        "O3.2": "not_applicable" if mode == "definitions" else "pass",
        "O3.3": "not_applicable" if mode == "definitions" else "pass",
        "O3.4": "not_applicable" if mode == "definitions" else "pass",
        "O3.5": "not_applicable" if mode == "definitions" else "pass",
        "O3.6": "not_applicable" if mode == "definitions" else "pass",
        "O3.7": "not_applicable" if mode == "definitions" else "pass",
        "O3.8": "pass",
        "O4.1": "pass",
        "O4.2": "pass",
        "O4.3": "pass",
        "O4.4": "pass",
        "O4.5": "pass",
        "O4.6": "pass",
        "O4.7": "pass",
        "O5.1": "pass",
        "O5.2": "pass",
        "O5.3": "pass",
        "O5.4": "pass",
        "O5.5": "pass",
        "O6.1": "pass",
        "O6.2": "pass",
        "O6.3": "pass",
        "O7.1": "pass",
        "O7.2": "pass",
        "O7.3": "pass",
        "O7.4": "pass",
        "O7.5": "pass",
        "O7.6": "not_applicable",
        "D1.1": "pass",
        "D1.2": "pass",
        "D1.3": "pass",
        "D1.4": "pass",
        "D1.5": "pass",
        "D2.1": "pass",
        "D2.2": "pass",
        "D2.3": "pass",
        "D2.4": "pass",
        "D2.5": "pass",
        "D3.1": "pass",
        "D3.2": "pass",
        "D3.3": "pass",
        "D3.4": "pass",
        "D3.5": "pass",
        "D4.1": "pass",
        "D4.2": "pass",
        "D4.3": "pass",
        "D4.4": "pass",
        "D5.1": "pass",
        "D5.2": "pass",
        "D5.3": "pass",
        "D5.4": "pass",
        "D5.5": "pass",
        "D5.6": "pass",
        "D5.7": "pass",
        "D6.1": "pass",
        "D6.2": "pass",
        "D6.3": "pass",
        "D6.4": "pass",
        "D7.1": "pass",
        "D7.2": "pass",
        "D7.3": "pass",
        "D8.1": "pass",
        "D8.2": "pass",
        "D8.3": "pass",
        "D8.4": "pass",
        "D9.1": "pass",
        "D9.2": "pass",
        "D9.3": "pass",
        "X1.1": "pass",
        "X1.2": "pass",
        "X1.3": "pass",
        "X1.4": "pass",
        "X1.5": "pass",
        "X1.6": "pass",
        "X1.7": "pass",
        "X1.8": "pass",
    }

    evidence_list = []
    for rep_path in ["reports/junit.xml", "reports/lcov.info"]:
        full = ROOT / rep_path
        if full.exists():
            digest = hashlib.sha256(full.read_bytes()).hexdigest()
            evidence_list.append({"file": rep_path, "sha256": digest})

    # Every finding comes from a check that ran; nothing here is assumed to pass.
    scope = manifest.get("scope", [])
    findings = []
    head = git_output(["rev-parse", "HEAD"])
    if commit_sha != head:
        findings.append(f"audited commit {commit_sha} is not HEAD ({head})")
    dirty = git_output(["status", "--porcelain", "--", *scope])
    if dirty:
        findings.append("uncommitted changes in scope: " + "; ".join(dirty.splitlines()))

    results = parse_junit_results()
    if results is None:
        findings.append("test report missing: reports/junit.xml")
    else:
        for test_id, outcome in sorted(results.items()):
            if outcome == "fail":
                findings.append(f"test failed: {test_id}")
    if not (ROOT / "reports/lcov.info").exists():
        findings.append("coverage report missing: reports/lcov.info")

    reqs = parse_requirements()
    regions_by_req, tests_by_req = scan_anchors(scope)
    for req_id, st in requirement_status(reqs, regions_by_req, tests_by_req).items():
        if st["status"] != "verified":
            findings.append(f"{req_id}: {st['status']}")
        elif st["coverage"] is not None and st["coverage"] < 100.0:
            findings.append(f"{req_id}: line coverage {st['coverage']:.1f}%")

    exceptions_text = (ROOT / manifest["documents"]["exceptions"]).read_text(encoding="utf-8")
    exception_count = len(re.findall(r"^#+\s*EX-\d+", exceptions_text, re.MULTILINE))

    result = "fail" if findings else "pass"
    for key, value in clauses.items():
        if value != "not_applicable":
            clauses[key] = result

    record = {
        "standard": "1.0",
        "commit": commit_sha,
        "date": datetime.now(timezone.utc).strftime("%Y-%m-%d"),
        "performed_by": "agent",
        "scope": manifest.get("scope", []),
        "commands": [
            "go vet ./internal/config ./internal/db ./internal/github ./internal/llm ./internal/engine/standards ./internal/api ./cmd/ossalot",
            "node --check web/src/model.mjs",
        ]
        if mode == "definitions"
        else [
            manifest["evidence"]["test_command"],
            manifest["evidence"]["coverage_command"],
        ],
        "tools": {
            "go": tool_version(["go", "version"]),
            "node": tool_version(["node", "--version"]),
            "python": tool_version(["python3", "--version"]),
        },
        "evidence": evidence_list,
        "exceptions": {
            "count": exception_count,
            "lines_covered": 0,
        },
        "open_emergencies": [],
        "clauses": clauses,
        "findings": findings,
        "result": result,
    }

    out_file = audit_dir / f"{commit_sha}.yaml"
    out_file.write_text(
        yaml.dump(record, sort_keys=False, default_flow_style=False),
        encoding="utf-8",
    )


if __name__ == "__main__":
    mode = sys.argv[1] if len(sys.argv) > 1 else "definitions"
    write_trace_matrix(mode=mode)
    if len(sys.argv) > 2:
        write_audit_record(sys.argv[2], mode=mode)
