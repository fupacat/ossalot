#!/usr/bin/env python3
"""Offline JUnit XML and LCOV Coverage Evidence Runner for OSSAlot (Go + Node)."""
import hashlib
import json
import os
import re
import shutil
import subprocess
import sys
import xml.etree.ElementTree as ET
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
REPORTS = ROOT / "reports"
GO_BIN = shutil.which("go") or "/home/eric/.gemini/antigravity/brain/dc8f5867-fc68-42d2-94ac-0a94391363ac/scratch/go/bin/go"
MODULE_PREFIX = "github.com/eric/ossalot/"
BCOV_MODULE = "github.com/fupacat/bcov"
BCOV_VERSION = "v0.1.0"
JS_NAME_SPECIALS = re.compile(r"[\\^$.*+?()\[\]{}|/]")


def get_env():
    env = os.environ.copy()
    go_dir = str(Path(GO_BIN).parent)
    env["PATH"] = f"{go_dir}:{env.get('PATH', '')}"
    return env


def bcov_env():
    """Environment for fetching the pinned bcov module from its private repository.

    Go fetches private modules through git. When the GitHub CLI is installed, its login is used as the
    git credential helper for this process only, so no global git configuration changes.
    """
    env = get_env()
    env["GOPRIVATE"] = BCOV_MODULE.rsplit("/", 1)[0] + "/*"
    env["GIT_TERMINAL_PROMPT"] = "0"
    if "GIT_CONFIG_COUNT" not in os.environ and shutil.which("gh"):
        env["GIT_CONFIG_COUNT"] = "1"
        env["GIT_CONFIG_KEY_0"] = "credential.https://github.com.helper"
        env["GIT_CONFIG_VALUE_0"] = "!gh auth git-credential"
    return env


def per_req_key(test_ids):
    """Stable name for one requirement's per-requirement evidence, derived from its test IDs."""
    return hashlib.sha256("\n".join(sorted(test_ids)).encode("utf-8")).hexdigest()[:16]


def map_go_tests_to_files():
    mapping = {}
    test_re = re.compile(r"^func\s+(Test\w+)\s*\(", re.MULTILINE)
    for path in ROOT.rglob("*_test.go"):
        rel = path.relative_to(ROOT).as_posix()
        pkg_rel = path.parent.relative_to(ROOT).as_posix()
        pkg = f"{MODULE_PREFIX}{pkg_rel}"
        text = path.read_text(encoding="utf-8")
        for m in test_re.finditer(text):
            mapping[(pkg, m.group(1))] = rel
    return mapping


def run_node_file_junit(js_rel):
    """Run one Node test file with the built-in junit reporter and return its real per-test results."""
    out_dir = REPORTS / "js"
    out_dir.mkdir(parents=True, exist_ok=True)
    out_file = out_dir / (js_rel.replace("/", "_") + ".xml")
    out_file.unlink(missing_ok=True)
    subprocess.run(
        [
            "node",
            "--test",
            "--test-reporter=junit",
            f"--test-reporter-destination={out_file}",
            js_rel,
        ],
        cwd=ROOT,
        env=get_env(),
        capture_output=True,
        text=True,
    )
    cases = []
    for tc in ET.parse(out_file).getroot().iter("testcase"):
        failed = tc.find("failure") is not None or tc.find("error") is not None
        cases.append(
            {
                "file": js_rel,
                "classname": js_rel,
                "name": tc.attrib.get("name", ""),
                "status": "fail" if failed else "pass",
                "time": tc.attrib.get("time", "0"),
            }
        )
    return cases


def run_tests():
    REPORTS.mkdir(parents=True, exist_ok=True)
    # Remove the previous test report so a failed run cannot leave stale evidence behind
    (REPORTS / "junit.xml").unlink(missing_ok=True)
    shutil.rmtree(REPORTS / "js", ignore_errors=True)
    test_file_map = map_go_tests_to_files()

    proc = subprocess.run(
        [GO_BIN, "test", "-json", "./..."],
        cwd=ROOT,
        env=get_env(),
        capture_output=True,
        text=True,
        check=True,
    )

    testcases = []
    for line in proc.stdout.splitlines():
        if not line.strip():
            continue
        ev = json.loads(line)
        test_name = ev.get("Test")
        action = ev.get("Action")
        pkg = ev.get("Package", "")
        if test_name and action in ("pass", "fail", "skip"):
            file_rel = test_file_map.get((pkg, test_name), "")
            testcases.append(
                {
                    "file": file_rel,
                    "classname": pkg,
                    "name": test_name,
                    "status": action,
                    "time": str(ev.get("Elapsed", 0.0)),
                }
            )

    # Run each web/src/*.test.mjs file on its own so every result keeps its file name
    for js_rel in sorted(p.relative_to(ROOT).as_posix() for p in (ROOT / "web/src").glob("*.test.mjs")):
        testcases.extend(run_node_file_junit(js_rel))

    suites = ET.Element("testsuites", attrib={"tests": str(len(testcases))})
    suite = ET.SubElement(
        suites,
        "testsuite",
        attrib={"name": "ossalot", "tests": str(len(testcases))},
    )
    for tc in testcases:
        el = ET.SubElement(
            suite,
            "testcase",
            attrib={
                "file": tc["file"],
                "classname": tc["classname"],
                "name": tc["name"],
                "time": tc["time"],
            },
        )
        if tc["status"] == "fail":
            ET.SubElement(el, "failure", attrib={"message": "test failed"})
        elif tc["status"] == "skip":
            ET.SubElement(el, "skipped")

    tree = ET.ElementTree(suites)
    ET.indent(tree, space="  ")
    tree.write(REPORTS / "junit.xml", encoding="utf-8", xml_declaration=True)


def convert_coverprofile_to_lcov(coverprofile_path, lcov_path):
    """Convert a Go coverprofile to LCOV line records. Go reports statements, so no branch records are written."""
    blocks_by_file = {}
    raw = coverprofile_path.read_text(encoding="utf-8").splitlines()
    block_re = re.compile(r"^(.+?):(\d+)\.\d+,(\d+)\.\d+\s+(\d+)\s+(\d+)$")
    for line in raw[1:]:
        m = block_re.match(line.strip())
        if not m:
            continue
        pkg_file, start_l, end_l, num_stmt, count = m.groups()
        rel_file = pkg_file.removeprefix(MODULE_PREFIX)
        blocks_by_file.setdefault(rel_file, []).append(
            (int(start_l), int(end_l), int(num_stmt), int(count))
        )

    lcov_lines = []
    for rel_file, blocks in sorted(blocks_by_file.items()):
        lcov_lines.append(f"SF:{rel_file}")
        line_counts = {}
        for s_line, e_line, _, count in blocks:
            for ln in range(s_line, e_line + 1):
                line_counts[ln] = max(line_counts.get(ln, 0), count)
        for ln, cnt in sorted(line_counts.items()):
            lcov_lines.append(f"DA:{ln},{cnt}")
        lf = len(line_counts)
        lh = sum(1 for c in line_counts.values() if c > 0)
        lcov_lines.append(f"LF:{lf}")
        lcov_lines.append(f"LH:{lh}")
        lcov_lines.append("end_of_record")

    lcov_path.write_text("\n".join(lcov_lines) + "\n", encoding="utf-8")


def drop_test_files(lcov_text):
    """Node also instruments the test files; only source files belong in the report."""
    keep = []
    skip = False
    for line in lcov_text.splitlines():
        if line.startswith("SF:"):
            skip = line.endswith(".test.mjs")
        if not skip:
            keep.append(line)
    return "\n".join(keep) + "\n"


def run_coverage():
    REPORTS.mkdir(parents=True, exist_ok=True)
    # Remove previous coverage reports; junit.xml belongs to the test command
    for name in ("coverage.out", "lcov.info", "js-lcov.info"):
        (REPORTS / name).unlink(missing_ok=True)
    cov_out = REPORTS / "coverage.out"
    subprocess.run(
        [
            GO_BIN,
            "test",
            "-covermode=count",
            f"-coverprofile={cov_out}",
            "./...",
        ],
        cwd=ROOT,
        env=get_env(),
        check=True,
    )
    # JavaScript line coverage comes from Node's own instrumentation, not synthesized
    js_test_files = sorted(
        p.relative_to(ROOT).as_posix() for p in (ROOT / "web/src").glob("*.test.mjs")
    )
    js_lcov = REPORTS / "js-lcov.info"
    subprocess.run(
        [
            "node",
            "--test",
            "--experimental-test-coverage",
            "--test-reporter=lcov",
            f"--test-reporter-destination={js_lcov}",
            *js_test_files,
        ],
        cwd=ROOT,
        env=get_env(),
        check=True,
    )
    convert_coverprofile_to_lcov(cov_out, REPORTS / "lcov.info")
    with (REPORTS / "lcov.info").open("a", encoding="utf-8") as combined:
        combined.write(drop_test_files(js_lcov.read_text(encoding="utf-8")))
    run_go_branch_coverage(REPORTS / "lcov.info")


def run_go_branch_coverage(lcov_path):
    """Count every outcome of every Go decision and add the results to lcov_path as BRDA records.

    bcov instruments copies of the in-scope packages under reports/_bcov. The copies are applied
    with -overlay, so the repository tree is not modified. The leading underscore keeps ./... from
    picking up the copies as packages.
    """
    out = REPORTS / "_bcov"
    shutil.rmtree(out, ignore_errors=True)
    subprocess.run(
        [GO_BIN, "run", f"{BCOV_MODULE}@{BCOV_VERSION}", "-root", str(ROOT), "-out", str(out)],
        cwd=ROOT,
        env=bcov_env(),
        check=True,
    )
    hits_dir = out / "hits"
    hits_dir.mkdir(parents=True, exist_ok=True)
    subprocess.run(
        [GO_BIN, "test", "-vet=off", "-count=1", "-overlay", str(out / "overlay.json"), "./..."],
        cwd=ROOT,
        env={**get_env(), "BCOV_OUT": str(hits_dir)},
        check=True,
    )

    manifest = json.loads((out / "decisions.json").read_text(encoding="utf-8"))
    branches_by_file = {}
    for pkg in manifest["packages"]:
        hit_file = hits_dir / pkg["hits"]
        if not hit_file.exists():
            raise RuntimeError(f"no branch hit counts for {pkg['dir']}: the test hook did not run")
        hits = {}
        for line in hit_file.read_text(encoding="utf-8").splitlines():
            index, count = line.split()
            hits[int(index)] = int(count)
        for d in pkg["decisions"]:
            for i in range(d["branches"]):
                taken = hits[d["base"] + i]
                branches_by_file.setdefault(d["file"], []).append((d["line"], d["base"], i, taken))

    records = lcov_path.read_text(encoding="utf-8").split("end_of_record\n")
    merged = []
    for record in records:
        if not record.strip():
            continue
        lines = record.splitlines()
        file_name = next((l[3:] for l in lines if l.startswith("SF:")), None)
        extra = branches_by_file.get(file_name, [])
        if extra:
            lines = [l for l in lines if not l.startswith(("BRDA:", "BRF:", "BRH:"))]
            brda = [f"BRDA:{ln},{block},{branch},{taken}" for ln, block, branch, taken in extra]
            hit = sum(1 for *_, taken in extra if taken > 0)
            lines += brda + [f"BRF:{len(extra)}", f"BRH:{hit}"]
        merged.append("\n".join(lines + ["end_of_record"]) + "\n")
    lcov_path.write_text("".join(merged), encoding="utf-8")


def run_per_req(test_ids):
    """Run only the given tests with coverage; write reports/per-req/<key>.lcov for them.

    The file name comes from per_req_key(), so the audit can find each requirement's evidence.
    A failing test makes the command exit non-zero and leaves no per-requirement file behind.
    """
    out_dir = REPORTS / "per-req"
    out_dir.mkdir(parents=True, exist_ok=True)
    key = per_req_key(test_ids)
    combined = out_dir / f"{key}.lcov"
    combined.unlink(missing_ok=True)

    go_names = {}
    js_names = {}
    for tid in test_ids:
        file_part, classname, name = tid.split("::", 2)
        if file_part.endswith(".mjs"):
            js_names.setdefault(file_part, []).append(name)
        else:
            go_names.setdefault(classname.removeprefix(MODULE_PREFIX), []).append(name)

    parts = []
    for index, (pkg_dir, names) in enumerate(sorted(go_names.items())):
        profile = out_dir / f"{key}-go{index}.out"
        pattern = "^(?:" + "|".join(re.escape(n) for n in names) + ")$"
        subprocess.run(
            [GO_BIN, "test", "-covermode=count", f"-coverprofile={profile}", "-run", pattern, f"./{pkg_dir}"],
            cwd=ROOT,
            env=get_env(),
            check=True,
        )
        lcov_part = out_dir / f"{key}-go{index}.lcov"
        convert_coverprofile_to_lcov(profile, lcov_part)
        parts.append(lcov_part.read_text(encoding="utf-8"))
        profile.unlink()
        lcov_part.unlink()

    for index, (js_rel, names) in enumerate(sorted(js_names.items())):
        dest = out_dir / f"{key}-js{index}.lcov"
        pattern = "^(?:" + "|".join(JS_NAME_SPECIALS.sub(r"\\\g<0>", n) for n in names) + ")$"
        subprocess.run(
            [
                "node",
                "--test",
                "--experimental-test-coverage",
                "--test-reporter=lcov",
                f"--test-reporter-destination={dest}",
                f"--test-name-pattern={pattern}",
                js_rel,
            ],
            cwd=ROOT,
            env=get_env(),
            check=True,
        )
        parts.append(drop_test_files(dest.read_text(encoding="utf-8")))
        dest.unlink()

    combined.write_text("".join(parts), encoding="utf-8")


if __name__ == "__main__":
    cmd = sys.argv[1] if len(sys.argv) > 1 else "test"
    if cmd == "test":
        run_tests()
    elif cmd == "coverage":
        run_coverage()
    elif cmd == "per-req":
        run_per_req(sys.argv[2:])
