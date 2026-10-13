// TRACEABILITY_MAPPING: REQ-019
export function buildTrafficSeries(snapshots) {
  if (!Array.isArray(snapshots) || snapshots.length === 0) {
    return [];
  }
  const sorted = [...snapshots].sort((a, b) =>
    String(a.snapshot_date).localeCompare(String(b.snapshot_date))
  );
  return sorted.map((s) => {
    const downloadsObj = s.registry_downloads || {};
    let registryTotal = 0;
    for (const val of Object.values(downloadsObj)) {
      registryTotal += Number(val || 0);
    }
    return {
      date: s.snapshot_date,
      stars: Number(s.stars || 0),
      viewsUnique: Number(s.views_unique || 0),
      clonesUnique: Number(s.clones_unique || 0),
      registryDownloads: registryTotal,
      adoptionScore: Number(s.adoption_score || 0),
    };
  });
}

// TRACEABILITY_MAPPING: REQ-019
export function aggregateReferrers(snapshots) {
  if (!Array.isArray(snapshots) || snapshots.length === 0) {
    return [];
  }
  const bySource = new Map();
  for (const snap of snapshots) {
    const refs = Array.isArray(snap.referrers) ? snap.referrers : [];
    for (const item of refs) {
      const existing = bySource.get(item.referrer) || {
        referrer: item.referrer,
        count: 0,
        uniques: 0,
      };
      existing.count += Number(item.count || 0);
      existing.uniques += Number(item.uniques || 0);
      bySource.set(item.referrer, existing);
    }
  }
  return Array.from(bySource.values()).sort((a, b) => b.count - a.count);
}
