import test from "node:test";
import assert from "node:assert/strict";
import { evaluateSetupCapabilities, computeSetupProgress } from "./setup_model.mjs";

// TRACEABILITY_MAPPING: REQ-034
test("[REQ-034] evaluateSetupCapabilities and computeSetupProgress", () => {
  // 1. Empty settings test
  const emptyCaps = evaluateSetupCapabilities({});
  assert.equal(emptyCaps.base_inspection.status, "degraded");
  assert.equal(emptyCaps.standard_remediation.status, "gated");
  assert.equal(emptyCaps.workflow_remediation.status, "gated");
  assert.equal(emptyCaps.sonarqube_analysis.status, "gated");
  assert.equal(emptyCaps.llm_synthesis.status, "fallback_mode");
  assert.equal(emptyCaps.webhook_alerts.status, "disabled");

  const emptyProgress = computeSetupProgress(emptyCaps);
  assert.equal(emptyProgress.completed, 0);
  assert.equal(emptyProgress.total, 6);
  assert.equal(emptyProgress.percentage, 0);

  // 2. Full settings test
  const fullCaps = evaluateSetupCapabilities({
    github_token: "ghp_validToken",
    sonar_token: "sqp_validSonar",
    sonar_host_url: "https://sonarcloud.io",
    sonar_org: "fupacat",
    llm_api_key: "sk-validKey",
    llm_provider: "openai",
    slack_webhook_url: "https://hooks.slack.com/services/123",
  });

  assert.equal(fullCaps.base_inspection.status, "enabled");
  assert.equal(fullCaps.standard_remediation.status, "enabled");
  assert.equal(fullCaps.workflow_remediation.status, "enabled");
  assert.equal(fullCaps.sonarqube_analysis.status, "enabled");
  assert.equal(fullCaps.llm_synthesis.status, "enabled");
  assert.equal(fullCaps.webhook_alerts.status, "enabled");

  const fullProgress = computeSetupProgress(fullCaps);
  assert.equal(fullProgress.completed, 6);
  assert.equal(fullProgress.percentage, 100);

  // 3. Null settings edge case
  const nullProgress = computeSetupProgress(null);
  assert.equal(nullProgress.percentage, 0);
});
