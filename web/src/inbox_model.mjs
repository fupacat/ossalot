// TRACEABILITY_MAPPING: REQ-031
export function groupInboxByStatus(opportunities) {
  const buckets = {
    new: [],
    saved: [],
    drafted: [],
    submitted: [],
    dismissed: [],
  };
  if (!Array.isArray(opportunities)) {
    return buckets;
  }
  for (const opp of opportunities) {
    const status = opp && buckets[opp.status] ? opp.status : "new";
    buckets[status].push(opp);
  }
  return buckets;
}

// TRACEABILITY_MAPPING: REQ-031
export function computeCampaignAttribution(opportunity, beforeSnapshot, afterSnapshot) {
  if (!opportunity || !beforeSnapshot || !afterSnapshot) {
    throw new Error("opportunity, beforeSnapshot, and afterSnapshot are required");
  }
  const starsDelta = Number(afterSnapshot.stars || 0) - Number(beforeSnapshot.stars || 0);
  const viewsDelta = Number(afterSnapshot.views_unique || 0) - Number(beforeSnapshot.views_unique || 0);
  const clonesDelta = Number(afterSnapshot.clones_unique || 0) - Number(beforeSnapshot.clones_unique || 0);

  return {
    opportunityId: opportunity.id || "",
    sourceCommunity: opportunity.source_community || "",
    submittedUrl: opportunity.submitted_url || "",
    starsDelta,
    viewsDelta,
    clonesDelta,
  };
}
