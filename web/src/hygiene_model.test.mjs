import test from "node:test";
import assert from "node:assert/strict";
import { generateHygieneConfigs } from "./hygiene_model.mjs";

// TRACEABILITY_MAPPING: REQ-036
test("[REQ-036] generateHygieneConfigs across ecosystems", () => {
  // 1. Go repository
  const goConfigs = generateHygieneConfigs("Go", "fupacat/ossalot");
  assert.equal(goConfigs.editorconfig.filename, ".editorconfig");
  assert.equal(goConfigs.gitattributes.filename, ".gitattributes");
  assert.equal(goConfigs.pre_commit.filename, ".pre-commit-config.yaml");
  assert.equal(goConfigs.linter.filename, ".golangci.yml");
  assert.match(goConfigs.linter.content, /errcheck/);

  // 2. Python repository
  const pyConfigs = generateHygieneConfigs("Python", "fupacat/girdle");
  assert.equal(pyConfigs.linter.filename, "ruff.toml");
  assert.match(pyConfigs.linter.content, /line-length = 88/);

  // 3. Java repository
  const javaConfigs = generateHygieneConfigs("Java", "fupacat/tsql-scriptdom-java");
  assert.equal(javaConfigs.linter.filename, "spotless.xml");
  assert.match(javaConfigs.linter.content, /googleJavaFormat/);

  // 4. Default / TS repository
  const defaultConfigs = generateHygieneConfigs(null, null);
  assert.equal(defaultConfigs.linter.filename, ".prettierrc.json");
});
