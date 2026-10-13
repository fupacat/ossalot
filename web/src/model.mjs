// TRACEABILITY_MAPPING: REQ-013
export function summarizePortfolio(repos) {
  if (!Array.isArray(repos) || repos.length === 0) {
    return {
      totalRepos: 0,
      avgStandardsScore: 0,
      ownedRepos: 0,
      failingChecksCount: 0,
    };
  }
  let scoreSum = 0;
  let ownedRepos = 0;
  let failingChecksCount = 0;

  for (const repo of repos) {
    scoreSum += Number(repo.standards_score || 0);
    if (repo.is_owned) {
      ownedRepos += 1;
    }
    const checks = repo.latest_audit && Array.isArray(repo.latest_audit.checks)
      ? repo.latest_audit.checks
      : [];
    for (const chk of checks) {
      if (chk.status === "fail") {
        failingChecksCount += 1;
      }
    }
  }

  return {
    totalRepos: repos.length,
    avgStandardsScore: Math.round(scoreSum / repos.length),
    ownedRepos,
    failingChecksCount,
  };
}

// TRACEABILITY_MAPPING: REQ-013
export function getStatusBadge(status) {
  if (status === "pass") {
    return { label: "PASS", tone: "emerald" };
  }
  if (status === "warn") {
    return { label: "WARN", tone: "amber" };
  }
  return { label: "FAIL", tone: "rose" };
}

// TRACEABILITY_MAPPING: REQ-013
export function buildRemediationRequest(checkKey, createPR) {
  if (!checkKey || typeof checkKey !== "string") {
    throw new Error("checkKey is required");
  }
  return {
    check_key: checkKey.trim(),
    create_pr: Boolean(createPR),
  };
}
