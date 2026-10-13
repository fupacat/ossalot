#!/usr/bin/env python3
"""Offline JUnit XML and LCOV Coverage Evidence Runner for OSSAlot (Go + Node)."""
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


def get_env():
    env = os.environ.copy()
    go_dir = str(Path(GO_BIN).parent)
    env["PATH"] = f"{go_dir}:{env.get('PATH', '')}"
    return env


def map_go_tests_to_files():
    mapping = {}
    test_re = re.compile(r"^func\s+(Test\w+)\s*\(", re.MULTILINE)
    for path in ROOT.rglob("*_test.go"):
        rel = path.relative_to(ROOT).as_posix()
        pkg_rel = path.parent.relative_to(ROOT).as_posix()
        pkg = f"github.com/eric/ossalot/{pkg_rel}"
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
    # Remove the previous report so a failed run cannot leave stale evidence behind
    (REPORTS / "junit.xml").unlink(missing_ok=True)
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
    blocks_by_file = {}
    raw = coverprofile_path.read_text(encoding="utf-8").splitlines()
    block_re = re.compile(r"^(.+?):(\d+)\.\d+,(\d+)\.\d+\s+(\d+)\s+(\d+)$")
    for line in raw[1:]:
        m = block_re.match(line.strip())
        if not m:
            continue
        pkg_file, start_l, end_l, num_stmt, count = m.groups()
        rel_file = pkg_file.removeprefix("github.com/eric/ossalot/")
        blocks_by_file.setdefault(rel_file, []).append(
            (int(start_l), int(end_l), int(num_stmt), int(count))
        )

    lcov_lines = []
    for rel_file, blocks in sorted(blocks_by_file.items()):
        lcov_lines.append(f"SF:{rel_file}")
        line_counts = {}
        brda_entries = []
        for idx, (s_line, e_line, _, count) in enumerate(blocks):
            for ln in range(s_line, e_line + 1):
                line_counts[ln] = max(line_counts.get(ln, 0), count)
            brda_entries.append((s_line, idx, 0, count))
        for ln, cnt in sorted(line_counts.items()):
            lcov_lines.append(f"DA:{ln},{cnt}")
        lf = len(line_counts)
        lh = sum(1 for c in line_counts.values() if c > 0)
        lcov_lines.append(f"LF:{lf}")
        lcov_lines.append(f"LH:{lh}")
        for s_line, blk_id, br_id, cnt in brda_entries:
            lcov_lines.append(f"BRDA:{s_line},{blk_id},{br_id},{cnt}")
        brf = len(brda_entries)
        brh = sum(1 for _, _, _, c in brda_entries if c > 0)
        lcov_lines.append(f"BRF:{brf}")
        lcov_lines.append(f"BRH:{brh}")
        lcov_lines.append("end_of_record")

    lcov_path.write_text("\n".join(lcov_lines) + "\n", encoding="utf-8")


def run_coverage():
    REPORTS.mkdir(parents=True, exist_ok=True)
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
    js_lcov.unlink(missing_ok=True)
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
    # Node also instruments the test files; only source files belong in the report
    keep = []
    skip = False
    for line in js_lcov.read_text(encoding="utf-8").splitlines():
        if line.startswith("SF:"):
            skip = line.endswith(".test.mjs")
        if not skip:
            keep.append(line)
    with (REPORTS / "lcov.info").open("a", encoding="utf-8") as combined:
        combined.write("\n".join(keep) + "\n")


def run_per_req(test_ids):
    REPORTS.mkdir(parents=True, exist_ok=True)
    for tid in test_ids:
        parts = tid.split("::")
        if parts[0].endswith(".mjs"):
            subprocess.run(
                ["node", "--test", parts[0]],
                cwd=ROOT,
                env=get_env(),
                check=True,
            )
        else:
            pkg = parts[1]
            func_name = parts[2]
            cov_out = REPORTS / f"cov_{func_name}.out"
            subprocess.run(
                [
                    GO_BIN,
                    "test",
                    "-covermode=count",
                    f"-coverprofile={cov_out}",
                    "-run",
                    f"^{func_name}$",
                    pkg,
                ],
                cwd=ROOT,
                env=get_env(),
                check=True,
            )


if __name__ == "__main__":
    cmd = sys.argv[1] if len(sys.argv) > 1 else "test"
    if cmd == "test":
        run_tests()
    elif cmd == "coverage":
        run_coverage()
    elif cmd == "per-req":
        run_per_req(sys.argv[2:])
