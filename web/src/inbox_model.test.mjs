import test from "node:test";
import assert from "node:assert/strict";
import {
  groupInboxByStatus,
  computeCampaignAttribution,
} from "./inbox_model.mjs";

// TRACEABILITY_MAPPING: REQ-031
test("[REQ-031] groupInboxByStatus and computeCampaignAttribution", () => {
  const emptyBuckets = groupInboxByStatus(null);
  assert.deepEqual(emptyBuckets, {
    new: [],
    saved: [],
    drafted: [],
    submitted: [],
    dismissed: [],
  });

  const grouped = groupInboxByStatus([
    { id: "opp-1", status: "new" },
    { id: "opp-2", status: "saved" },
    { id: "opp-3", status: "drafted" },
    { id: "opp-4", status: "submitted" },
    { id: "opp-5", status: "dismissed" },
    { id: "opp-6", status: "unknown" },
  ]);
  assert.equal(grouped.new.length, 2);
  assert.equal(grouped.saved.length, 1);
  assert.equal(grouped.drafted.length, 1);
  assert.equal(grouped.submitted.length, 1);
  assert.equal(grouped.dismissed.length, 1);

  const attr = computeCampaignAttribution(
    {
      id: "opp-4",
      source_community: "Hacker News (Show HN)",
      submitted_url: "https://news.ycombinator.com/item?id=1",
    },
    { stars: 100, views_unique: 50, clones_unique: 10 },
    { stars: 125, views_unique: 190, clones_unique: 42 }
  );
  assert.deepEqual(attr, {
    opportunityId: "opp-4",
    sourceCommunity: "Hacker News (Show HN)",
    submittedUrl: "https://news.ycombinator.com/item?id=1",
    starsDelta: 25,
    viewsDelta: 140,
    clonesDelta: 32,
  });

  const zeroAttr = computeCampaignAttribution({}, {}, {});
  assert.equal(zeroAttr.starsDelta, 0);

  assert.throws(
    () => computeCampaignAttribution(null, {}, {}),
    /opportunity, beforeSnapshot, and afterSnapshot are required/
  );
});
