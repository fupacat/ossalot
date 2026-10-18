import test from "node:test";
import assert from "node:assert/strict";
import {
  validateRepoConfig,
  normalizeRepoConfig,
  composeAgentPrompt,
  copilotWorkflowPresets,
} from "./repo_config_model.mjs";
import { PIPELINE_TEMPLATES } from "./pipeline_model.mjs";

// TRACEABILITY_MAPPING: REQ-056
test("[REQ-056] validateRepoConfig handles invalid config types", () => {
  assert.equal(validateRepoConfig(null).valid, false);
  assert.equal(validateRepoConfig(undefined).valid, false);
  assert.equal(validateRepoConfig("string").valid, false);
  assert.equal(validateRepoConfig(123).valid, false);
});

// TRACEABILITY_MAPPING: REQ-056
test("[REQ-056] validateRepoConfig validates base_branch rules", () => {
  assert.equal(validateRepoConfig({ base_branch: "main" }).valid, true);
  assert.equal(validateRepoConfig({ base_branch: "" }).valid, true);
  assert.equal(validateRepoConfig({ base_branch: "main branch" }).valid, false);
  assert.equal(validateRepoConfig({ base_branch: "main..dev" }).valid, false);
  assert.equal(validateRepoConfig({ base_branch: "/main" }).valid, false);
  assert.equal(validateRepoConfig({ base_branch: "main/" }).valid, false);
});

// TRACEABILITY_MAPPING: REQ-056
test("[REQ-056] validateRepoConfig validates copilot_handle and branch_prefix", () => {
  assert.equal(validateRepoConfig({ copilot_handle: "@copilot" }).valid, true);
  assert.equal(validateRepoConfig({ copilot_handle: "" }).valid, true);
  assert.equal(validateRepoConfig({ copilot_handle: "copilot" }).valid, false);
  assert.equal(validateRepoConfig({ copilot_handle: "@copilot agent" }).valid, false);

  assert.equal(validateRepoConfig({ branch_prefix: "feat/" }).valid, true);
  assert.equal(validateRepoConfig({ branch_prefix: "" }).valid, true);
  assert.equal(validateRepoConfig({ branch_prefix: "copilot prefix" }).valid, false);
});

// TRACEABILITY_MAPPING: REQ-056
test("[REQ-056] normalizeRepoConfig applies expected defaults", () => {
  const def = normalizeRepoConfig(null);
  assert.equal(def.default_branch, "main");
  assert.equal(def.base_branch, "main");
  assert.equal(def.branch_prefix, "copilot/");
  assert.equal(def.copilot_handle, "@copilot");
  assert.equal(def.issue_labels, "copilot, remediation");
  assert.equal(def.copilot_prompt, "");

  const customized = normalizeRepoConfig({
    default_branch: "trunk",
    base_branch: "develop",
    branch_prefix: "agent/",
    copilot_handle: "@custom-bot",
    issue_labels: "bot, auto",
    copilot_prompt: "  Run tests before PR  ",
  });
  assert.equal(customized.default_branch, "trunk");
  assert.equal(customized.base_branch, "develop");
  assert.equal(customized.branch_prefix, "agent/");
  assert.equal(customized.copilot_handle, "@custom-bot");
  assert.equal(customized.issue_labels, "bot, auto");
  assert.equal(customized.copilot_prompt, "Run tests before PR");

  const fallbackBase = normalizeRepoConfig({
    default_branch: "master",
  });
  assert.equal(fallbackBase.base_branch, "master");
});

// TRACEABILITY_MAPPING: REQ-056
test("[REQ-056] composeAgentPrompt produces structured prompts with defaults and overrides", () => {
  const minimalPrompt = composeAgentPrompt();
  assert.ok(minimalPrompt.includes("@copilot"));
  assert.ok(minimalPrompt.includes("Target Base Branch"));
  assert.ok(minimalPrompt.includes("`main`"));
  assert.ok(minimalPrompt.includes("copilot/"));

  const fullRepo = {
    name: "tsql-scriptdom-java",
    full_name: "fupacat/tsql-scriptdom-java",
    base_branch: "trunk",
    branch_prefix: "remediate/",
    copilot_handle: "@workspace",
    copilot_prompt: "Always execute 'mvn clean test'.",
  };

  const detailedPrompt = composeAgentPrompt(fullRepo, {
    title: "Configure SonarCloud CI",
    details: "Missing sonar analysis workflow in .github/workflows/sonar.yml",
    specificationYaml: "name: Sonar\non: push",
  });

  assert.ok(detailedPrompt.startsWith("@workspace please address this task for fupacat/tsql-scriptdom-java."));
  assert.ok(detailedPrompt.includes("Objective: Configure SonarCloud CI"));
  assert.ok(detailedPrompt.includes("Missing sonar analysis workflow"));
  assert.ok(detailedPrompt.includes("```yaml\nname: Sonar\non: push\n```"));
  assert.ok(detailedPrompt.includes("Target Base Branch**: Open any pull request against `trunk`"));
  assert.ok(detailedPrompt.includes("Working Branch Prefix**: Name your working branch with `remediate/`"));
  assert.ok(detailedPrompt.includes("Custom Repository Rules**: Always execute 'mvn clean test'."));

  const nameOnlyRepo = { name: "simple-repo" };
  const namePrompt = composeAgentPrompt(nameOnlyRepo);
  assert.ok(namePrompt.includes("address this task for simple-repo."));
});

// TRACEABILITY_MAPPING: REQ-056
test("[REQ-056] copilotWorkflowPresets lists the dispatchable handler workflows", () => {
  const presets = copilotWorkflowPresets();
  assert.deepEqual(Object.keys(presets), ["ci_failure_handler", "review_handler"]);
  assert.equal(presets.ci_failure_handler.id, "copilot_ci_failure_handler");
  assert.equal(presets.ci_failure_handler.filename, ".github/workflows/copilot-ci-failure-handler.yml");
  assert.equal(presets.review_handler.id, "copilot_review_handler");
  assert.equal(presets.review_handler.filename, ".github/workflows/copilot-review-handler.yml");
  for (const wf of Object.values(presets)) {
    assert.equal(wf.name, PIPELINE_TEMPLATES[wf.id].name);
    assert.equal(wf.filename, PIPELINE_TEMPLATES[wf.id].filename);
  }
});
