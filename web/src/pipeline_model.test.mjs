import test from "node:test";
import assert from "node:assert/strict";
import { generatePipelineWorkflow, PIPELINE_TEMPLATES } from "./pipeline_model.mjs";

// TRACEABILITY_MAPPING: REQ-035
test("[REQ-035] generatePipelineWorkflow and PIPELINE_TEMPLATES", () => {
  assert.equal(Object.keys(PIPELINE_TEMPLATES).length, 5);

  const basic = generatePipelineWorkflow("basic_ci", "fupacat/ossalot");
  assert.equal(basic.filename, ".github/workflows/ci.yml");
  assert.match(basic.content, /name: "CI Pipeline"/);

  const sonar = generatePipelineWorkflow("sonar_analysis", "fupacat/tsql-scriptdom-java");
  assert.equal(sonar.filename, ".github/workflows/sonar.yml");
  assert.equal(sonar.required_secret, "SONAR_TOKEN");
  assert.match(sonar.content, /SONAR_TOKEN/);

  const copilotCi = generatePipelineWorkflow("copilot_ci_failure_handler", "fupacat/ossalot");
  assert.equal(copilotCi.filename, ".github/workflows/copilot-ci-failure-handler.yml");
  assert.match(copilotCi.content, /@copilot The workflow/);

  const copilotReview = generatePipelineWorkflow("copilot_review_handler", "fupacat/ossalot");
  assert.equal(copilotReview.filename, ".github/workflows/copilot-review-handler.yml");
  assert.match(copilotReview.content, /@copilot Changes were requested/);

  const prAuto = generatePipelineWorkflow("pr_automation", "fupacat/ossalot");
  assert.equal(prAuto.filename, ".github/workflows/pr-automation.yml");
  assert.match(prAuto.content, /gh pr merge/);

  // Fallback branch
  const fallback = generatePipelineWorkflow("unknown_type", null);
  assert.equal(fallback.filename, ".github/workflows/ci.yml");
});
