import test from "node:test";
import assert from "node:assert/strict";
import { buildTrafficSeries, aggregateReferrers } from "./metrics_model.mjs";

// TRACEABILITY_MAPPING: REQ-019
test("[REQ-019] buildTrafficSeries and aggregateReferrers", () => {
  assert.deepEqual(buildTrafficSeries([]), []);
  assert.deepEqual(aggregateReferrers([]), []);

  const rawSnapshots = [
    {
      snapshot_date: "2026-10-12",
      stars: 120,
      views_unique: 50,
      clones_unique: 15,
      registry_downloads: { npm: 300, pypi: 200 },
      adoption_score: 78,
      referrers: [
        { referrer: "reddit.com", count: 20, uniques: 15 },
        { referrer: "news.ycombinator.com", count: 60, uniques: 50 },
      ],
    },
    {
      snapshot_date: "2026-10-11",
      stars: 0,
      views_unique: 0,
      clones_unique: 0,
      registry_downloads: null,
      adoption_score: 0,
      referrers: null,
    },
    {
      snapshot_date: "2026-10-10",
      stars: 95,
      views_unique: 30,
      clones_unique: 8,
      registry_downloads: { npm: 100, broken: 0 },
      adoption_score: 60,
      referrers: [
        { referrer: "news.ycombinator.com", count: 40, uniques: 35 },
        { referrer: "lobste.rs", count: 0, uniques: 0 },
      ],
    },
  ];

  const series = buildTrafficSeries(rawSnapshots);
  assert.equal(series.length, 3);
  assert.equal(series[0].date, "2026-10-10");
  assert.equal(series[2].date, "2026-10-12");
  assert.equal(series[2].registryDownloads, 500);

  const refs = aggregateReferrers(rawSnapshots);
  assert.deepEqual(refs, [
    { referrer: "news.ycombinator.com", count: 100, uniques: 85 },
    { referrer: "reddit.com", count: 20, uniques: 15 },
    { referrer: "lobste.rs", count: 0, uniques: 0 },
  ]);
});
