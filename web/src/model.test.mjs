import test from "node:test";
import assert from "node:assert/strict";
import {
  summarizePortfolio,
  getStatusBadge,
  buildRemediationRequest,
} from "./model.mjs";

// TRACEABILITY_MAPPING: REQ-013
test("[REQ-013] summarizePortfolio, getStatusBadge, and buildRemediationRequest", () => {
  assert.deepEqual(summarizePortfolio([]), {
    totalRepos: 0,
    avgStandardsScore: 0,
    ownedRepos: 0,
    failingChecksCount: 0,
  });

  const summary = summarizePortfolio([
    {
      standards_score: 90,
      is_owned: true,
      latest_audit: {
        checks: [
          { check_key: "license_valid", status: "pass" },
          { check_key: "security_policy", status: "fail" },
        ],
      },
    },
    {
      standards_score: 70,
      is_owned: false,
      latest_audit: null,
    },
    {
      standards_score: 0,
      is_owned: false,
      latest_audit: {
        checks: [{ check_key: "readme_comprehensive", status: "fail" }],
      },
    },
  ]);

  assert.deepEqual(summary, {
    totalRepos: 3,
    avgStandardsScore: 53,
    ownedRepos: 1,
    failingChecksCount: 2,
  });

  assert.deepEqual(getStatusBadge("pass"), { label: "PASS", tone: "emerald" });
  assert.deepEqual(getStatusBadge("warn"), { label: "WARN", tone: "amber" });
  assert.deepEqual(getStatusBadge("fail"), { label: "FAIL", tone: "rose" });

  assert.deepEqual(buildRemediationRequest(" contributing_guide ", true), {
    check_key: "contributing_guide",
    create_pr: true,
  });
  assert.throws(() => buildRemediationRequest("", false), /checkKey is required/);
});
