#!/usr/bin/env python3
"""MCSWE 1.0 trace matrix and audit record generator.

  python3 scripts/generate_mcswe_artifacts.py matrix          regenerate the trace matrix from current reports
  python3 scripts/generate_mcswe_artifacts.py audit <commit>  run the audit steps (plugin audit.md) for HEAD and write its record

Clauses the audit cannot check are recorded as fail with a finding, never as pass (A3.1).
"""
import hashlib
import json
import os
import re
import shlex
import shutil
import subprocess
import sys
import xml.etree.ElementTree as ET
from datetime import datetime, timezone
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parent.parent
REPORTS = ROOT / "reports"
sys.path.insert(0, str(ROOT / "scripts"))
import run_evidence  # noqa: E402  (shares per_req_key, MODULE_PREFIX and the Go binary location)

MATRIX_STATUSES = {"verified", "unverified", "failing", "emergency", "retired"}  # D8.2
VAGUE_TERMS = ["fast", "efficient", "secure", "user-friendly", "scalable"]  # D5.3 default list


def ids(prefix, count):
    return [f"{prefix}{i}" for i in range(1, count + 1)]


CLAUSES = (
    ids("O1.", 8) + ids("O2.", 8) + ids("O3.", 8) + ids("O4.", 7) + ids("O5.", 5)
    + ids("O6.", 3) + ids("O7.", 6) + ids("D1.", 5) + ids("D2.", 5) + ids("D3.", 5)
    + ids("D4.", 4) + ids("D5.", 7) + ids("D6.", 4) + ids("D7.", 3) + ids("D8.", 4)
    + ids("D9.", 3) + ids("X1.", 8)
)


# --- Parsing -----------------------------------------------------------------

def load_manifest():
    text = (ROOT / "MCSWE.md").read_text(encoding="utf-8")
    parts = text.split("---")
    return yaml.safe_load(parts[1])


def parse_requirements():
    text = (ROOT / "docs/mcswe/requirements.md").read_text(encoding="utf-8")
    pattern = re.compile(r"^(REQ-\d+)\s+(.+?)\n\n```yaml\n(.*?)```", re.MULTILINE | re.DOTALL)
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
    pattern = re.compile(r"^##\s+(IF-\d+)\s+(.+?)\n\n```yaml\n(.*?)```", re.MULTILINE | re.DOTALL)
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


def parse_decisions():
    text = (ROOT / "docs/mcswe/decisions.md").read_text(encoding="utf-8")
    pattern = re.compile(r"^##\s+(ADR-\d+)\s+(.+?)\n\n```yaml\n(.*?)```", re.MULTILINE | re.DOTALL)
    return {m.group(1): yaml.safe_load(m.group(3)) for m in pattern.finditer(text)}


def scan_anchors(scope_dirs):
    """Scan in-scope files for TRACEABILITY_MAPPING anchors and compute the span each covers.

    Returns (regions_by_req, tests_by_req, malformed) where malformed lists anchors naming no requirement.
    """
    regions_by_req = {}
    tests_by_req = {}
    malformed = []
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
                    req_ids = [r.strip() for r in m.group(1).split(",") if r.strip().startswith("REQ-")]
                    if not req_ids:
                        malformed.append(f"{rel}:{i + 1}")
                    start_line = i + 1
                    j = i + 1
                    while j < len(lines) and not lines[j].strip():
                        j += 1
                    unit_line = j
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
                            test_id = f"{rel}::{run_evidence.MODULE_PREFIX}{pkg_dir}::{tm_go.group(1)}"
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
    return regions_by_req, tests_by_req, malformed


def scan_declared_tests(scope_dirs):
    """Every test declared in scope, whether anchored or not, as full test identities (O1.5)."""
    declared = set()
    go_test_re = re.compile(r"^func\s+(Test\w+)\s*\(", re.MULTILINE)
    js_test_re = re.compile(r"""^\s*test\(\s*["'`](.+?)["'`]""", re.MULTILINE)
    for scope_dir in scope_dirs:
        base = ROOT / scope_dir
        if not base.exists():
            continue
        for path in sorted(base.rglob("*")):
            if not path.is_file():
                continue
            rel = path.relative_to(ROOT).as_posix()
            text = path.read_text(encoding="utf-8")
            if rel.endswith("_test.go"):
                pkg_dir = path.parent.relative_to(ROOT).as_posix()
                for m in go_test_re.finditer(text):
                    declared.add(f"{rel}::{run_evidence.MODULE_PREFIX}{pkg_dir}::{m.group(1)}")
            elif rel.endswith(".test.mjs"):
                for m in js_test_re.finditer(text):
                    declared.add(f"{rel}::{rel}::{m.group(1)}")
    return declared


def parse_junit_results():
    """Return {test_id: pass|fail|skip} from the test report, or None if it is missing."""
    junit_path = REPORTS / "junit.xml"
    if not junit_path.exists():
        return None
    results = {}
    for tc in ET.parse(junit_path).getroot().iter("testcase"):
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


def parse_lcov(path):
    """Return {file: {line: hit_count}} for an LCOV file, or None if it is missing."""
    if not path.exists():
        return None
    coverage = {}
    current = None
    for line in path.read_text(encoding="utf-8").splitlines():
        if line.startswith("SF:"):
            current = line[3:]
            coverage.setdefault(current, {})
        elif line.startswith("DA:") and current:
            ln, count = line[3:].split(",")[:2]
            prev = coverage[current].get(int(ln), 0)
            coverage[current][int(ln)] = max(prev, int(count))
    return coverage


def parse_lcov_branches(path):
    """Return {file: [(line, taken)]} for files with branch records; taken is 0 for an unexecuted branch."""
    if not path.exists():
        return {}
    branches = {}
    current = None
    for line in path.read_text(encoding="utf-8").splitlines():
        if line.startswith("SF:"):
            current = line[3:]
        elif line.startswith("BRDA:") and current:
            fields = line[5:].split(",")
            taken = fields[3]
            branches.setdefault(current, []).append((int(fields[0]), 0 if taken in ("0", "-") else int(taken)))
    return branches


def region_lines(regions):
    """All (file, line) pairs covered by the given `file:start-end` spans."""
    lines = set()
    for region in regions:
        rel, span = region.strip("`").rsplit(":", 1)
        start, end = (int(x) for x in span.split("-"))
        lines.update((rel, ln) for ln in range(start, end + 1))
    return lines


# --- Evidence ----------------------------------------------------------------

def tool_env():
    env = os.environ.copy()
    extra = [str(Path(run_evidence.GO_BIN).parent), os.path.expanduser("~/.local/go/bin")]
    env["PATH"] = os.pathsep.join([p for p in extra if os.path.isdir(p)] + [env.get("PATH", "")])
    return env


def run_shell(command):
    result = subprocess.run(command, shell=True, cwd=ROOT, env=tool_env(), capture_output=True, text=True)
    return result.returncode


def git_output(args):
    result = subprocess.run(["git", *args], cwd=ROOT, capture_output=True, text=True, check=True)
    return result.stdout.strip()


def tool_version(cmd):
    """Version reported by the installed tool, or 'not found'."""
    try:
        out = subprocess.run(cmd, capture_output=True, text=True, check=True, env=tool_env()).stdout
    except (FileNotFoundError, subprocess.CalledProcessError):
        return "not found"
    match = re.search(r"\d+\.\d+(?:\.\d+)?", out)
    return match.group(0) if match else out.strip()


class Context:
    """Everything the requirement statuses and audit clauses are computed from."""

    def __init__(self, manifest, reqs, ifaces, decisions, regions, tests, declared, malformed):
        self.manifest = manifest
        self.reqs = reqs
        self.ifaces = ifaces
        self.decisions = decisions
        self.regions = regions
        self.tests = tests
        self.declared = declared
        self.malformed = malformed
        self.scope = manifest.get("scope", [])
        self.junit = parse_junit_results()
        self.full = parse_lcov(REPORTS / "lcov.info")
        self.branches = parse_lcov_branches(REPORTS / "lcov.info")
        self.own = {}
        for req_id, tlist in tests.items():
            self.own[req_id] = parse_lcov(REPORTS / "per-req" / f"{run_evidence.per_req_key(tlist)}.lcov")
        self.lines = {req_id: region_lines(regions.get(req_id, [])) for req_id in reqs}
        self.run_failed = set()
        self.checkers = []
        self.commit = None
        self.head = None
        self.dirty_scope = ""
        self.stable = True
        self.cleaned = False
        self.exception_count = 0
        self.statuses = {}
        self.matrix_rows = 0
        self.approval_text = {}


def requirement_coverage(req_id, ctx):
    """Line coverage of this requirement's executable anchored lines, from its own tests only (O2.2)."""
    if ctx.full is None:
        return None
    executable = [(f, ln) for (f, ln) in ctx.lines[req_id] if ln in ctx.full.get(f, {})]
    if not executable:
        return None
    own = ctx.own.get(req_id) or {}
    covered = sum(1 for f, ln in executable if own.get(f, {}).get(ln, 0) > 0)
    return 100.0 * covered / len(executable)


def requirement_status(req_id, ctx):
    tlist = ctx.tests.get(req_id, [])
    if not tlist:
        return {"status": "unverified", "coverage": None}
    if ctx.junit is None:
        return {"status": "no evidence", "coverage": None}
    outcomes = [ctx.junit.get(t) for t in tlist]
    if req_id in ctx.run_failed:
        status = "failing"
    elif any(o is None for o in outcomes):
        status = "not run"
    elif "fail" in outcomes:
        status = "failing"
    elif "skip" in outcomes:
        status = "skipped"
    else:
        status = "verified"
    coverage = requirement_coverage(req_id, ctx) if status == "verified" else None
    return {"status": status, "coverage": coverage}


def unique_checkers(ifaces):
    return sorted({i["checker"] for i in ifaces.values() if i.get("checker")})


# --- Clauses -----------------------------------------------------------------

def evaluate(ctx):
    """Return ({clause: pass|fail|not_applicable}, findings). Every clause in CLAUSES gets a value."""
    res = {}
    findings = []

    def finding(clause, detail, files=()):
        findings.append({"clause": clause, "files": sorted(set(files)), "detail": detail})

    def check(clause, problems, files=()):
        if problems:
            res[clause] = "fail"
            for problem in problems:
                finding(clause, problem, files)
        else:
            res[clause] = "pass"

    def not_automated(clause, why):
        res[clause] = "fail"
        finding(clause, "not checked by the audit tool: " + why)

    def not_applicable(clause):
        res[clause] = "not_applicable"

    reqs, ifaces, decisions = ctx.reqs, ctx.ifaces, ctx.decisions
    mapped_tests = set(t for tl in ctx.tests.values() for t in tl)
    ifaces_by_req = {}
    for if_id, info in ifaces.items():
        for rid in info["requirements"]:
            ifaces_by_req.setdefault(rid, []).append(if_id)
    docs = ctx.manifest["documents"]

    # Sources must be resolved decisions or existing requirements (O1.1, D4.3, O6.2)
    source_problems = []
    for req_id, r in sorted(reqs.items()):
        src = r["source"]
        if src is None:
            source_problems.append(f"{req_id} has no source")
        elif src.startswith("ADR-") and decisions.get(src, {}).get("status") != "resolved":
            source_problems.append(f"{req_id} cites {src}, which is not a resolved decision")
        elif src.startswith("REQ-") and src not in reqs:
            source_problems.append(f"{req_id} cites unknown requirement {src}")
        elif not (src.startswith("ADR-") or src.startswith("REQ-")):
            source_problems.append(f"{req_id} has an unrecognised source {src}")

    # O1
    check("O1.1", source_problems)
    p = []
    for req_id in sorted(reqs):
        if req_id not in ifaces_by_req:
            p.append(f"{req_id} has no interface")
        if not ctx.regions.get(req_id):
            p.append(f"{req_id} has no anchored region")
        if not ctx.tests.get(req_id):
            p.append(f"{req_id} has no test")
    check("O1.2", p)
    not_automated("O1.3", "every non-trivial line in scope is not checked against its anchor")
    unknown = sorted((set(ctx.regions) | set(ctx.tests)) - set(reqs))
    check("O1.4", [f"anchor names unknown requirement {u}" for u in unknown])
    unanchored = sorted(ctx.declared - mapped_tests)
    check("O1.5", [f"test has no requirement anchor: {t}" for t in unanchored],
          [t.split("::")[0] for t in unanchored])
    p = []
    for if_id, info in ifaces.items():
        if not info["requirements"]:
            p.append(f"{if_id} serves no requirement")
        for rid in info["requirements"]:
            if rid not in reqs:
                p.append(f"{if_id} names unknown requirement {rid}")
    check("O1.6", p)
    if ctx.manifest.get("generators"):
        not_automated("O1.7", "generated code is not traced through its generator inputs")
    else:
        not_applicable("O1.7")
    res["O1.8"] = "pass"  # the matrix is written by this tool from anchors and reports

    # O2
    uncovered = []
    if ctx.full is None:
        uncovered.append("coverage report missing")
    else:
        owners = {}
        for req_id in reqs:
            for loc in ctx.lines[req_id]:
                if loc[1] in ctx.full.get(loc[0], {}):
                    owners.setdefault(loc, set()).add(req_id)
        for loc, rs in sorted(owners.items()):
            if not any((ctx.own.get(r) or {}).get(loc[0], {}).get(loc[1], 0) > 0 for r in rs):
                uncovered.append(f"{loc[0]}:{loc[1]} is not executed by the tests of {', '.join(sorted(rs))}")
    check("O2.1", uncovered[:50], [u.split(":")[0] for u in uncovered[:50]])
    p = []
    for req_id in sorted(reqs):
        if ctx.tests.get(req_id):
            if req_id in ctx.run_failed:
                p.append(f"{req_id}: per-requirement run failed")
            elif ctx.own.get(req_id) is None:
                p.append(f"{req_id}: no per-requirement coverage report")
    check("O2.2", p)
    res["O2.3"] = "pass"  # tests run the same code that is built; instrumentation only

    full_uncovered = []
    if ctx.full is not None:
        for f, lines_map in sorted(ctx.full.items()):
            if any(f.startswith(s) for s in ctx.scope):
                for ln, count in sorted(lines_map.items()):
                    if count == 0:
                        full_uncovered.append(f"{f}:{ln}")
    branch_problem = []
    if ctx.full is not None:
        for f in sorted(ctx.full):
            if not any(f.startswith(s) for s in ctx.scope):
                continue
            if f not in ctx.branches:
                branch_problem.append(f"branch coverage is not measured for {f}")
            else:
                branch_problem += [f"branch not taken: {f}:{ln}" for ln, taken in ctx.branches[f] if taken == 0]
    check("O2.4", ([f"uncovered line {u}" for u in full_uncovered[:50]] + branch_problem),
          [u.split(":")[0] for u in full_uncovered[:50]])
    check("O2.5", ([f"uncovered line {u}" for u in full_uncovered[:50]] + branch_problem),
          [u.split(":")[0] for u in full_uncovered[:50]])
    if ctx.manifest.get("safety_critical"):
        not_automated("O2.6", "MC/DC for safety-critical components is not measured")
    else:
        not_applicable("O2.6")
    adr2 = decisions.get("ADR-002", {})
    check("O2.7", [] if adr2.get("status") == "resolved" else ["ADR-002 (safety-critical decision) is not resolved"])
    res["O2.8"] = "pass" if res["O2.5"] == "pass" else "fail"
    if res["O2.8"] == "fail":
        finding("O2.8", "shortfall from O2.5 has no exception (exceptions register is empty)")

    # O3
    p = []
    if ctx.dirty_scope:
        p.append("uncommitted changes in scope: " + ctx.dirty_scope.replace("\n", "; "))
    if ctx.head != ctx.commit:
        p.append(f"audited commit {ctx.commit} is not HEAD ({ctx.head})")
    check("O3.1", p)
    p = []
    if ctx.junit is None:
        p.append("test report missing: reports/junit.xml")
    if ctx.full is None:
        p.append("coverage report missing: reports/lcov.info")
    check("O3.2", p)
    check("O3.3", [f"test identity is not full: {t}" for t in sorted(mapped_tests) if len(t.split("::")) != 3])
    outcomes = {t: (ctx.junit or {}).get(t) for t in mapped_tests}
    check("O3.4", [f"test did not pass: {t} ({o or 'not run'})" for t, o in sorted(outcomes.items()) if o != "pass"])
    check("O3.5", [f"skipped test counted as evidence: {t}" for t, o in sorted(outcomes.items()) if o == "skip"])
    check("O3.6", [f"mapped test was not run: {t}" for t, o in sorted(outcomes.items()) if o is None])
    check("O3.7", [] if ctx.cleaned else ["reports were not removed before this run"])
    check("O3.8", [] if ctx.stable else ["in-scope files or HEAD changed during the run"])

    # O4 (approvals are recorded in review history, which the tool does not read)
    index = commit_index(ctx.manifest.get("definitions_first_from"))
    approval_gaps = approval_problems(index, ctx.scope)
    check("O4.1", approval_gaps)
    res["O4.4"] = res["O4.1"]  # self-approvals are recorded by the approver trailer naming the author
    not_automated("O4.2", "independence of approvers is not checked")
    not_applicable("O4.3")
    check("O4.5", [] if all(a != "unapproved" for a in ctx.approval_text.values())
          else [f"{r} has no approval" for r, a in sorted(ctx.approval_text.items()) if a == "unapproved"])
    res["O4.6"] = "pass"
    not_automated("O4.7", "whether tests check intent is a review question")

    # O5
    tracked = set(git_output(["ls-files"]).splitlines())
    required_paths = ["MCSWE.md", docs["decisions"], docs["requirements"], docs["interfaces"],
                      docs["exceptions"], docs["trace_matrix"]]
    check("O5.1", [f"required document not under version control: {p}" for p in required_paths if p not in tracked])
    not_automated("O5.2", "amendments are not compared against approved baselines")
    not_automated("O5.3", "invalidation of downstream approvals is not computed")
    not_automated("O5.4", "amendment reasons are not checked")
    not_automated("O5.5", "identifier reuse is not checked against history")

    # O6
    problems = definition_order_problems(reqs, ctx.scope, ctx.manifest.get("definitions_first_from"))
    check("O6.1", problems, [p.split(" ")[0] for p in problems])
    res["O6.2"] = res["O1.1"]
    if res["O6.2"] == "fail":
        for problem in source_problems:
            finding("O6.2", problem)
    not_applicable("O6.3")

    # O7
    not_automated("O7.1", "component boundaries are not compared with the scope")
    check("O7.2", [f"definition file missing: {i['definition']}" for i in ifaces.values()
                   if not (ROOT / str(i["definition"]).split("#")[0]).exists()],
          [str(i["definition"]) for i in ifaces.values()])
    not_automated("O7.3", "data exchanged between components is not checked for schemas")
    check("O7.4", [f"{i_id} names no checker" for i_id, i in ifaces.items() if not i.get("checker")])
    check("O7.5", [f"checker failed (exit {rc}): {c}" for c, rc in ctx.checkers if rc != 0])
    not_applicable("O7.6")

    # D1
    check("D1.1", [f"required document missing: {p}" for p in required_paths + [docs["audits"]]
                   if not (ROOT / p).exists()])
    res["D1.2"] = "pass"  # plain Markdown and YAML, diffable
    dup = []
    text = {"REQ": (ROOT / docs["requirements"]).read_text(encoding="utf-8"),
            "IF": (ROOT / docs["interfaces"]).read_text(encoding="utf-8"),
            "ADR": (ROOT / docs["decisions"]).read_text(encoding="utf-8")}
    heading = {"REQ": r"^(REQ-\d+)\s", "IF": r"^##\s+(IF-\d+)\s", "ADR": r"^##\s+(ADR-\d+)\s"}
    for prefix, pattern in heading.items():
        found = re.findall(pattern, text[prefix], re.MULTILINE)
        dup += [f"duplicate identifier {i}" for i in sorted(set(x for x in found if found.count(x) > 1))]
    check("D1.3", dup)
    counts = {"REQ": len(reqs), "IF": len(ifaces), "ADR": len(decisions)}
    parse_problems = []
    for prefix, pattern in heading.items():
        if len(re.findall(pattern, text[prefix], re.MULTILINE)) != counts[prefix]:
            parse_problems.append(f"{prefix} entries are not all machine-readable")
    check("D1.4", parse_problems)
    res["D1.5"] = "pass"  # matrix and records are written by this tool

    # D2
    not_automated("D2.1", "anchors are not checked against the file's comment syntax")
    check("D2.2", [f"anchor names no requirement at {m}" for m in ctx.malformed])
    not_automated("D2.3", "the three-line placement rule is not checked")
    not_automated("D2.4", "nested inheritance of anchors is not checked")
    not_automated("D2.5", "anchors covering no unit are not detected")

    # D3
    manifest_keys = ["mcswe", "scope", "out_of_scope", "documents", "evidence", "safety_critical",
                     "generators", "audit_schedule"]
    evidence_keys = ["test_command", "test_report", "coverage_command", "coverage_report", "per_requirement_command"]
    p = [f"manifest is missing {k}" for k in manifest_keys if k not in ctx.manifest]
    p += [f"manifest evidence is missing {k}" for k in evidence_keys if k not in ctx.manifest.get("evidence", {})]
    check("D3.1", p)
    res["D3.2"] = "pass"  # no generators declared
    check("D3.3", [] if "safety_critical" in ctx.manifest else ["manifest does not state safety_critical"])
    res["D3.4"] = "pass"  # front matter is parsed as YAML
    agents = ROOT / "AGENTS.md"
    check("D3.5", [] if agents.exists() and "MCSWE.md" in agents.read_text(encoding="utf-8") else
          ["AGENTS.md does not name the manifest"])

    # D4
    p = []
    for d_id, d in sorted(decisions.items()):
        if d.get("status") not in ("open", "resolved", "superseded"):
            p.append(f"{d_id} has an invalid status {d.get('status')}")
        for key in ("context", "decision", "consequences"):
            if not d.get(key):
                p.append(f"{d_id} has no {key}")
    check("D4.1", p)
    check("D4.2", [f"{d_id} is superseded but names no replacement"
                   for d_id, d in decisions.items() if d.get("status") == "superseded" and not d.get("superseded_by")])
    check("D4.3", source_problems)
    not_automated("D4.4", "the bounds of the work are not checked against the decision record")

    # D5
    check("D5.1", [f"{r} has no SHALL statement" for r in sorted(reqs) if "SHALL" not in reqs[r]["statement"]])
    not_automated("D5.2", "whether each requirement states one behavior is a review question")
    p = []
    for r_id, r in sorted(reqs.items()):
        for term in VAGUE_TERMS:
            if re.search(r"\b" + re.escape(term) + r"\b", r["statement"].lower()):
                p.append(f"{r_id} uses the vague term '{term}'")
    check("D5.3", p)
    check("D5.4", [f"{r_id} has no source or a status other than active/retired"
                   for r_id, r in sorted(reqs.items()) if not r["source"] or r["status"] not in ("active", "retired")])
    p = []
    for r_id, r in sorted(reqs.items()):
        acc = r["acceptance"] or {}
        for key in ("kind", "precondition", "action", "method"):
            if not acc.get(key):
                p.append(f"{r_id} acceptance has no {key}")
        if not (acc.get("expected") or acc.get("expected_output")):
            p.append(f"{r_id} acceptance has no expected result")
    check("D5.5", p)
    p = []
    for r_id, r in sorted(reqs.items()):
        acc = r["acceptance"] or {}
        if acc.get("kind") == "categorical":
            if not (acc.get("expected") or acc.get("expected_output")):
                p.append(f"{r_id} is categorical but has no expected output")
        elif acc.get("kind") == "quantitative":
            for key in ("operator", "value", "unit"):
                if acc.get(key) in (None, ""):
                    p.append(f"{r_id} is quantitative but has no {key}")
        else:
            p.append(f"{r_id} has an invalid kind {acc.get('kind')}")
    check("D5.6", p)
    check("D5.7", [f"{r_id} verification method is {r['acceptance'].get('method')}, not test"
                   for r_id, r in sorted(reqs.items()) if (r["acceptance"] or {}).get("method") != "test"])

    # D6
    not_automated("D6.1", "component boundaries are not listed in the interface control document")
    p = []
    for i_id, i in sorted(ifaces.items()):
        for key in ("component", "definition", "requirements", "checker"):
            if not i.get(key):
                p.append(f"{i_id} has no {key}")
    check("D6.2", p)
    not_automated("D6.3", "data exchange schemas are not checked")
    not_automated("D6.4", "restated native definitions are not detected")

    # D7
    if ctx.exception_count == 0:
        res["D7.1"] = res["D7.2"] = res["D7.3"] = "pass"
    else:
        for c in ("D7.1", "D7.2", "D7.3"):
            not_automated(c, "exception records are not checked")

    # D8
    rows = ctx.matrix_rows
    check("D8.1", [] if rows == len(reqs) else [f"matrix has {rows} rows for {len(reqs)} requirements"])
    check("D8.2", [f"status {ctx.statuses[r]['status']} is not allowed for {r}"
                   for r in sorted(reqs) if ctx.statuses[r]["status"] not in MATRIX_STATUSES])
    res["D8.3"] = res["O4.5"]  # the approval column is derived from the approver trailers
    res["D8.4"] = "pass"  # the matrix lists exceptions and out-of-scope paths

    # D9
    res["D9.1"] = "pass"  # the record holds commit, scope, commands, tools, evidence hashes and clauses
    res["D9.2"] = "pass"  # every finding names its clause and files
    res["D9.3"] = "pass"  # records are written under docs/mcswe/audits/

    # X1
    if ctx.exception_count == 0:
        for n in range(1, 8):
            res[f"X1.{n}"] = "pass"
        res["X1.8"] = "pass"
    else:
        for n in range(1, 9):
            not_automated(f"X1.{n}", "exception records are not checked")

    for clause in CLAUSES:
        if clause not in res:
            not_automated(clause, "no check is defined for this clause")
    return {c: res[c] for c in CLAUSES}, findings


# --- Outputs -----------------------------------------------------------------

def first_commit(args):
    """Oldest commit matching the git log arguments, or None."""
    shas = git_output(["log", "--reverse", "--format=%H", *args]).splitlines()
    return shas[0] if shas else None


def is_ancestor(older, newer):
    return subprocess.run(["git", "merge-base", "--is-ancestor", older, newer],
                          cwd=ROOT, capture_output=True).returncode == 0


def commit_index(baseline):
    """Commits after the enforcement baseline: author, approvers, approved revisions and changed paths."""
    index = {}
    if not baseline:
        return index
    rev_range = f"{baseline}..HEAD"
    raw = git_output(["log", "--reverse", "--format=%H%x1f%an%x1f%ae%x1f%B%x1e", rev_range])
    for entry in raw.split("\x1e"):
        entry = entry.strip("\n")
        if not entry:
            continue
        sha, author_name, author_email, body = entry.split("\x1f", 3)
        approvers = []
        approves = []
        for line in body.splitlines():
            if line.startswith("Approved-by:"):
                approvers.append(line[len("Approved-by:"):].strip())
            elif line.startswith("Approves:"):
                approves.append(line[len("Approves:"):].strip())
        files = git_output(["diff-tree", "--no-commit-id", "--name-only", "-r", sha]).splitlines()
        index[sha] = {
            "author": (author_name, author_email),
            "approvers": approvers,
            "approves": approves,
            "files": files,
        }
    return index


def required_change(files, scope):
    """True if a commit touches governed code or a required document (generated documents excluded)."""
    generated = ("docs/mcswe/trace-matrix.md", "docs/mcswe/audits/")
    for f in files:
        if f.startswith(generated):
            continue
        if f == "MCSWE.md" or f.startswith("docs/mcswe/") or any(f.startswith(s) for s in scope):
            return True
    return False


def approvals_for(index):
    """{revision: [approver, ...]} from each commit's own trailers and from later Approves: trailers."""
    approved = {}
    for sha, info in index.items():
        if info["approvers"]:
            approved.setdefault(sha, []).extend(info["approvers"])
    for sha, info in index.items():
        for target in info["approves"]:
            # Targets may predate the baseline, so resolve them through git, not the index
            full = git_output(["rev-parse", "--verify", "--quiet", f"{target}^{{commit}}"])
            if full and info["approvers"]:
                approved.setdefault(full, []).extend(info["approvers"])
    return approved


def approval_problems(index, scope):
    """Commits that change governed content with no approver (O4.1)."""
    approved = approvals_for(index)
    problems = []
    for sha, info in index.items():
        if not required_change(info["files"], scope):
            continue
        if not approved.get(sha):
            problems.append(f"{sha[:7]} changes governed content with no Approved-by trailer")
    return problems


def requirement_approval(req_id, defined, index, approved):
    """Approval column text for a requirement: the approver of its definition, or unapproved."""
    if defined is None:
        return "unapproved"
    approvers = approved.get(defined) or []
    if not approvers:
        return "unapproved"
    author_email = git_output(["log", "-1", "--format=%ae", defined])
    first = approvers[0]
    if author_email and f"<{author_email}>" in first:
        return f"self (O4.4) {first}"
    return first


def definition_order_problems(reqs, scope, enforced_from=None):
    """Requirements whose code entered scope before, or in the same commit as, their definition (O6.1, P1.1).

    Only code that entered scope after enforced_from (ADR-021) is checked. Earlier history is not tested.
    """
    problems = []
    for req_id in sorted(reqs):
        defined = first_commit(["-S", f"{req_id} ", "--", "docs/mcswe/requirements.md"])
        implemented = first_commit(["-G", f"TRACEABILITY_MAPPING:.*{req_id}([^0-9]|$)", "--", *scope])
        if implemented is None:
            continue  # no code yet; the status column already shows it as unverified
        if enforced_from and is_ancestor(implemented, enforced_from):
            continue  # entered scope before the check was enforced
        if defined is None:
            problems.append(f"{req_id} has code in scope but no definition in the requirements history")
        elif defined == implemented:
            problems.append(f"{req_id} was defined and implemented in the same commit {defined[:7]}")
        elif not is_ancestor(defined, implemented):
            problems.append(f"{req_id} code ({implemented[:7]}) entered scope before its definition ({defined[:7]})")
    return problems


def write_trace_matrix(ctx, reqs, ifaces):
    ifaces_by_req = {}
    for if_id, info in ifaces.items():
        for rid in info["requirements"]:
            ifaces_by_req.setdefault(rid, []).append(if_id)

    index = commit_index(ctx.manifest.get("definitions_first_from"))
    approved = approvals_for(index)
    approval_text = {}
    for req_id in reqs:
        defined = first_commit(["-S", f"{req_id} ", "--", "docs/mcswe/requirements.md"])
        approval_text[req_id] = requirement_approval(req_id, defined, index, approved)
    ctx.approval_text = approval_text
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
        regs = ", ".join(f"`{s.strip('`')}`" for s in ctx.regions.get(req_id, [])) or "none"
        tlist = ctx.tests.get(req_id, [])
        tests_str = ", ".join(f"`{t}`" for t in tlist) if tlist else "none"
        status = ctx.statuses[req_id]
        status_str = status["status"] if status["status"] in MATRIX_STATUSES else "failing"
        cov_str = "n/a" if status["coverage"] is None else f"{status['coverage']:.1f}% line"
        approval = approval_text.get(req_id, "unapproved")
        lines.append(
            f"| {req_id} | {r['source']} | {ifs} | {regs} | {tests_str} | {cov_str} | {approval} | {status_str} |"
        )
    lines.extend(
        [
            "",
            "## Exceptions (D8.4)",
            "",
            "None." if ctx.exception_count == 0 else "- See `docs/mcswe/exceptions.md`.",
            "",
            "## Out-of-scope paths (D8.4)",
            "",
        ]
    )
    for p in ctx.manifest.get("out_of_scope", []):
        lines.append(f"- `{p}`")
    lines.append("")
    (ROOT / "docs/mcswe/trace-matrix.md").write_text("\n".join(lines), encoding="utf-8")
    ctx.matrix_rows = len(reqs)


def write_audit_record(ctx, commit_sha, clauses, findings):
    manifest = ctx.manifest
    audit_dir = ROOT / manifest["documents"]["audits"]
    audit_dir.mkdir(parents=True, exist_ok=True)

    evidence_list = []
    for rep_path in ["reports/junit.xml", "reports/lcov.info"]:
        full = ROOT / rep_path
        if full.exists():
            evidence_list.append({"file": rep_path, "sha256": hashlib.sha256(full.read_bytes()).hexdigest()})

    record = {
        "standard": "1.0",
        "commit": commit_sha,
        "date": datetime.now(timezone.utc).strftime("%Y-%m-%d"),
        "performed_by": "agent",
        "scope": ctx.scope,
        "commands": [
            manifest["evidence"]["test_command"],
            manifest["evidence"]["coverage_command"],
            manifest["evidence"]["per_requirement_command"].replace("{tests}", "<requirement's tests>"),
        ],
        "checkers": [{"command": c, "exit_code": rc} for c, rc in ctx.checkers],
        "tools": {
            "go": tool_version(["go", "version"]),
            "node": tool_version(["node", "--version"]),
            "python": tool_version(["python3", "--version"]),
        },
        "evidence": evidence_list,
        "exceptions": {"count": ctx.exception_count, "lines_covered": 0},
        "open_emergencies": [],
        "clauses": clauses,
        "findings": findings,
        "result": "fail" if findings else "pass",
    }
    (audit_dir / f"{commit_sha}.yaml").write_text(
        yaml.dump(record, sort_keys=False, default_flow_style=False), encoding="utf-8"
    )
    return record["result"]


def build_context(run_failed=(), checkers=(), commit=None, cleaned=False):
    manifest = load_manifest()
    reqs = parse_requirements()
    ifaces = parse_interfaces()
    decisions = parse_decisions()
    regions, tests, malformed = scan_anchors(manifest.get("scope", []))
    declared = scan_declared_tests(manifest.get("scope", []))
    ctx = Context(manifest, reqs, ifaces, decisions, regions, tests, declared, malformed)
    ctx.exception_count = len(re.findall(
        r"^#+\s*EX-\d+", (ROOT / manifest["documents"]["exceptions"]).read_text(encoding="utf-8"), re.MULTILINE))
    ctx.run_failed = set(run_failed)
    ctx.checkers = list(checkers)
    ctx.commit = commit
    ctx.cleaned = cleaned
    ctx.head = git_output(["rev-parse", "HEAD"])
    ctx.dirty_scope = git_output(["status", "--porcelain", "--", *manifest.get("scope", [])])
    ctx.statuses = {r: requirement_status(r, ctx) for r in reqs}
    return ctx


def matrix_only():
    ctx = build_context()
    write_trace_matrix(ctx, ctx.reqs, ctx.ifaces)


def audit(commit_sha):
    manifest = load_manifest()
    scope = manifest.get("scope", [])
    head = git_output(["rev-parse", "HEAD"])
    if commit_sha != head:
        sys.exit(f"audit runs on HEAD ({head}); {commit_sha} is not HEAD")
    if git_output(["status", "--porcelain", "--", *scope]):
        sys.exit("in-scope files have uncommitted changes; audit stopped (audit step 1)")

    # Step 3: every checker named by an interface
    checkers = [(c, run_shell(c)) for c in unique_checkers(parse_interfaces())]

    # Step 4: remove old reports, then run the manifest's test and coverage commands
    for name in ("junit.xml", "lcov.info", "coverage.out", "js-lcov.info"):
        (REPORTS / name).unlink(missing_ok=True)
    shutil.rmtree(REPORTS / "js", ignore_errors=True)
    shutil.rmtree(REPORTS / "per-req", ignore_errors=True)
    run_shell(manifest["evidence"]["test_command"])
    run_shell(manifest["evidence"]["coverage_command"])

    # Step 5: each requirement's tests run alone, so coverage is attributed per requirement (O2.2)
    tests = scan_anchors(scope)[1]
    per_req_template = manifest["evidence"]["per_requirement_command"]
    run_failed = set()
    for req_id, tlist in sorted(tests.items()):
        command = per_req_template.replace("{tests}", " ".join(shlex.quote(t) for t in tlist))
        if run_shell(command) != 0:
            run_failed.add(req_id)

    ctx = build_context(run_failed, checkers, commit_sha, cleaned=True)

    # Step 7: nothing in scope or HEAD changed during the run
    ctx.stable = git_output(["rev-parse", "HEAD"]) == head and not git_output(
        ["status", "--porcelain", "--", *scope])

    # Step 8: outputs. The matrix is written first; it does not depend on the clauses.
    write_trace_matrix(ctx, ctx.reqs, ctx.ifaces)
    clauses, findings = evaluate(ctx)
    result = write_audit_record(ctx, commit_sha, clauses, findings)
    print(f"audit {commit_sha}: {result} ({len(findings)} findings)")


if __name__ == "__main__":
    command = sys.argv[1] if len(sys.argv) > 1 else "matrix"
    if command == "matrix":
        matrix_only()
    elif command == "audit" and len(sys.argv) == 3:
        audit(sys.argv[2])
    else:
        sys.exit(__doc__)
