import test from "node:test";
import assert from "node:assert/strict";
import {
  filterOpportunities,
  formatPRPlanSummary,
} from "./discovery_model.mjs";

// TRACEABILITY_MAPPING: REQ-025
test("[REQ-025] filterOpportunities and formatPRPlanSummary", () => {
  assert.deepEqual(filterOpportunities([]), []);

  const opps = [
    {
      external_id: "gh:acme/low#1",
      category: "github_issue",
      status: "new",
      relevance_score: 65,
    },
    {
      external_id: "hn:100",
      category: "hackernews",
      status: "new",
      relevance_score: 85,
    },
    {
      external_id: "gh:acme/high#2",
      category: "github_issue",
      status: "saved",
      relevance_score: 92,
      source_community: "acme/high",
      pr_implementation_plan: "1. Add dependency\n2. Add unit test",
    },
    {
      external_id: "gh:acme/mid#3",
      category: "github_issue",
      status: "new",
      relevance_score: 78,
    },
  ];

  const filtered = filterOpportunities(opps, {
    minScore: 70,
    category: "github_issue",
    status: "saved",
  });
  assert.equal(filtered.length, 1);
  assert.equal(filtered[0].external_id, "gh:acme/high#2");

  const sortedAllHigh = filterOpportunities(opps, { minScore: 70 });
  assert.equal(sortedAllHigh.length, 3);
  assert.equal(sortedAllHigh[0].relevance_score, 92);

  const plan = formatPRPlanSummary(opps[2]);
  assert.deepEqual(plan, {
    externalId: "gh:acme/high#2",
    targetCommunity: "acme/high",
    relevanceScore: 92,
    stepsCount: 2,
    steps: ["1. Add dependency", "2. Add unit test"],
  });

  assert.throws(() => formatPRPlanSummary(null), /opportunity object is required/);
});

// TRACEABILITY_MAPPING: REQ-025
test("[REQ-025] filterOpportunities and formatPRPlanSummary tolerate missing fields", () => {
  const opps = [
    { external_id: "gh:acme/no-score#4", category: "github_issue", status: "new" },
    { external_id: "hn:200", category: "hackernews", status: "new", relevance_score: 80 },
  ];

  // Without a minimum score every opportunity passes; a missing score counts as 0 and sorts last
  const all = filterOpportunities(opps);
  assert.deepEqual(all.map((o) => o.external_id), ["hn:200", "gh:acme/no-score#4"]);

  // A minimum score excludes the opportunity that has no score
  const high = filterOpportunities(opps, { minScore: 50 });
  assert.deepEqual(high.map((o) => o.external_id), ["hn:200"]);

  // The sort must handle a missing score on either side of the comparison
  const reversed = filterOpportunities([opps[1], opps[0]]);
  assert.deepEqual(reversed.map((o) => o.external_id), ["hn:200", "gh:acme/no-score#4"]);
  const scoreLast = filterOpportunities([
    { external_id: "gh:acme/a#1", relevance_score: 10 },
    { external_id: "gh:acme/b#2" },
    { external_id: "gh:acme/c#3", relevance_score: 90 },
  ]);
  assert.deepEqual(scoreLast.map((o) => o.external_id), ["gh:acme/c#3", "gh:acme/a#1", "gh:acme/b#2"]);

  // Missing plan fields produce empty values rather than undefined
  assert.deepEqual(formatPRPlanSummary({}), {
    externalId: "",
    targetCommunity: "",
    relevanceScore: 0,
    stepsCount: 0,
    steps: [],
  });
});
