import test from "node:test";
import assert from "node:assert/strict";
import { evaluateSonarOnboarding } from "./sonar_model.mjs";

// TRACEABILITY_MAPPING: REQ-037
test("[REQ-037] evaluateSonarOnboarding with registered/unregistered orgs and imported/unimported projects", () => {
  // 1. Registered and imported project (both {component:{key}} and direct {key})
  const imported = evaluateSonarOnboarding(
    "fupacat/tsql-scriptdom-java",
    { organizations: [{ key: "fupacat" }] },
    { component: { key: "fupacat_tsql-scriptdom-java" } }
  );
  assert.equal(imported.status, "registered_and_imported");
  assert.equal(imported.orgExists, true);
  assert.equal(imported.projectExists, true);
  assert.match(imported.actionURL, /summary\/new_code\?id=fupacat_tsql-scriptdom-java/);
  assert.equal(imported.localCommunityEdition.defaultURL, "http://localhost:9000");

  const importedDirect = evaluateSonarOnboarding(
    "fupacat/ossalot",
    { organizations: [{ key: "fupacat" }] },
    { key: "fupacat_ossalot", name: "ossalot" }
  );
  assert.equal(importedDirect.status, "registered_and_imported");
  assert.equal(importedDirect.projectExists, true);

  // 2. Org exists, but project not yet imported
  const notImported = evaluateSonarOnboarding(
    "fupacat/ossalot",
    { organizations: [{ key: "fupacat" }] },
    { errors: [{ msg: "Component key not found" }] }
  );
  assert.equal(notImported.status, "registered_not_imported");
  assert.equal(notImported.orgExists, true);
  assert.equal(notImported.projectExists, false);
  assert.match(notImported.actionURL, /projects\/create\?org=fupacat/);

  // 3. Org not registered at all
  const unreg = evaluateSonarOnboarding(
    "newuser/newrepo",
    { organizations: [] },
    null
  );
  assert.equal(unreg.status, "not_registered");
  assert.equal(unreg.orgExists, false);
  assert.equal(unreg.projectExists, false);
  assert.match(unreg.actionURL, /sessions\/new/);

  // 4. Default coordinates edge case
  const fallback = evaluateSonarOnboarding(null, null, null);
  assert.equal(fallback.owner, "owner");
  assert.equal(fallback.status, "not_registered");
});
