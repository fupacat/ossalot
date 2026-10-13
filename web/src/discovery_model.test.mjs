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
