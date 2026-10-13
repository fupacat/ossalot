// TRACEABILITY_MAPPING: REQ-034
export function evaluateSetupCapabilities(settings) {
  const s = settings || {};
  const hasGhToken = Boolean(s.github_token && s.github_token.trim().length > 0);
  const hasLlmKey = Boolean(s.llm_api_key && s.llm_api_key.trim().length > 0);
  const hasSlack = Boolean(s.slack_webhook_url && s.slack_webhook_url.trim().length > 0);
  const hasDiscord = Boolean(s.discord_webhook_url && s.discord_webhook_url.trim().length > 0);
  const hasSonarToken = Boolean(s.sonar_token && s.sonar_token.trim().length > 0);

  return {
    base_inspection: {
      status: hasGhToken ? "enabled" : "degraded",
      details: hasGhToken
        ? "Authenticated GitHub API access active (5,000 req/hr rate limit)."
        : "Unauthenticated public access (60 req/hr rate limit; rate-limiting may occur).",
      required_secrets: ["github_token"],
      scope_hint: "public_repo or repo",
    },
    standard_remediation: {
      status: hasGhToken ? "enabled" : "gated",
      details: hasGhToken
        ? "1-Click branch, commit & Pull Request creation enabled on owned repositories."
        : "Requires GitHub Personal Access Token or OAuth token to commit and open pull requests.",
      required_secrets: ["github_token"],
      scope_hint: "repo (Full control of private/public repositories)",
    },
    workflow_remediation: {
      status: hasGhToken ? "enabled" : "gated",
      details: "Editing or creating files under .github/workflows/ requires explicit workflow OAuth scope.",
      required_secrets: ["github_token"],
      scope_hint: "workflow (Update GitHub Action workflows)",
    },
    sonarqube_analysis: {
      status: hasSonarToken ? "enabled" : "gated",
      details: hasSonarToken
        ? `Configured for ${s.sonar_host_url || "SonarCloud"} (Org: ${s.sonar_org || "default"}).`
        : "Optional: Configure SonarCloud/SonarQube User Token to gate PR quality and view code smells.",
      required_secrets: ["sonar_token"],
      scope_hint: "SonarQube User Token (Generate under User Account > Security > Tokens)",
    },
    llm_synthesis: {
      status: hasLlmKey ? "enabled" : "fallback_mode",
      details: hasLlmKey
        ? `Active with ${s.llm_provider || "gemini"} model (${s.llm_fast_model || "fast"}).`
        : "Deterministic repository-tailored templates active (no API key required).",
      required_secrets: ["llm_api_key"],
      scope_hint: "Provider API Key (Gemini, OpenAI, Anthropic, or Ollama endpoint)",
    },
    webhook_alerts: {
      status: hasSlack || hasDiscord ? "enabled" : "disabled",
      details: hasSlack || hasDiscord
        ? "Real-time alerts active for high-value opportunities (score >= 85%)."
        : "Optional: Configure Slack or Discord incoming webhook URL to receive instant alerts.",
      required_secrets: ["slack_webhook_url", "discord_webhook_url"],
      scope_hint: "Incoming Webhook URL",
    },
  };
}

// TRACEABILITY_MAPPING: REQ-034
export function computeSetupProgress(capabilities) {
  const caps = Object.values(capabilities || {});
  if (caps.length === 0) return { completed: 0, total: 0, percentage: 0 };
  const completed = caps.filter((c) => c.status === "enabled").length;
  return {
    completed,
    total: caps.length,
    percentage: Math.round((completed / caps.length) * 100),
  };
}
