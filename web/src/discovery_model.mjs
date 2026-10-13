// TRACEABILITY_MAPPING: REQ-025
export function filterOpportunities(opportunities, filters = {}) {
  if (!Array.isArray(opportunities) || opportunities.length === 0) {
    return [];
  }
  const minScore = Number(filters.minScore || 0);
  const category = filters.category || "";
  const status = filters.status || "";

  return opportunities
    .filter((opp) => {
      if (Number(opp.relevance_score || 0) < minScore) {
        return false;
      }
      if (category && opp.category !== category) {
        return false;
      }
      if (status && opp.status !== status) {
        return false;
      }
      return true;
    })
    .sort((a, b) => Number(b.relevance_score || 0) - Number(a.relevance_score || 0));
}

// TRACEABILITY_MAPPING: REQ-025
export function formatPRPlanSummary(opportunity) {
  if (!opportunity || typeof opportunity !== "object") {
    throw new Error("opportunity object is required");
  }
  const rawSteps = String(opportunity.pr_implementation_plan || "")
    .split("\n")
    .map((line) => line.trim())
    .filter(Boolean);

  return {
    externalId: opportunity.external_id || "",
    targetCommunity: opportunity.source_community || "",
    relevanceScore: Number(opportunity.relevance_score || 0),
    stepsCount: rawSteps.length,
    steps: rawSteps,
  };
}
