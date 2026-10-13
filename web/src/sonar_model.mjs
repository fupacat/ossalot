// TRACEABILITY_MAPPING: REQ-037
export function evaluateSonarOnboarding(repoFullName, orgSearchPayload, compSearchPayload) {
  const full = repoFullName || "owner/repo";
  const [owner, name] = full.includes("/") ? full.split("/") : ["owner", full];
  const projectKey = `${owner}_${name}`;

  const orgs = (orgSearchPayload && Array.isArray(orgSearchPayload.organizations))
    ? orgSearchPayload.organizations
    : [];
  const orgExists = orgs.some(
    (o) => o.key && o.key.toLowerCase() === owner.toLowerCase()
  );

  const comp = compSearchPayload && (compSearchPayload.component || compSearchPayload);
  const compKey = comp && (comp.key || "");
  const projectExists = Boolean(compKey && compKey.toLowerCase() === projectKey.toLowerCase());

  let status = "not_registered";
  let summary = "";
  let actionLabel = "";
  let actionURL = "";

  if (orgExists && projectExists) {
    status = "registered_and_imported";
    summary = `SonarCloud project active for ${full} under organization "${owner}".`;
    actionLabel = "View on SonarCloud";
    actionURL = `https://sonarcloud.io/summary/new_code?id=${projectKey}`;
  } else if (orgExists && !projectExists) {
    status = "registered_not_imported";
    summary = `Organization "${owner}" exists on SonarCloud, but repository "${name}" has not been imported yet.`;
    actionLabel = "Import Project to SonarCloud (Free)";
    actionURL = `https://sonarcloud.io/projects/create?org=${owner}`;
  } else {
    status = "not_registered";
    summary = `Organization "${owner}" is not yet registered on SonarCloud. Public open-source repositories qualify for 100% free analysis.`;
    actionLabel = "Register via GitHub OAuth (Free for Open Source)";
    actionURL = "https://sonarcloud.io/sessions/new?return_to=%2Fprojects%2Fcreate";
  }

  const localCommunityEdition = {
    title: "Local SonarQube Community Edition (Self-Hosted / On-Prem)",
    description: "Run SonarQube locally via Docker with zero cloud dependencies or paid plan requirements.",
    dockerCommand: "docker run -d --name sonarqube -p 9000:9000 -e SONAR_ES_BOOTSTRAP_CHECKS_DISABLE=true sonarqube:community",
    defaultURL: "http://localhost:9000",
    defaultCredentials: "admin / admin (prompted to reset on first login)",
    localScanHint: `sonar-scanner -Dsonar.projectKey=${projectKey} -Dsonar.sources=. -Dsonar.host.url=http://localhost:9000`,
  };

  return {
    repo: full,
    owner,
    projectKey,
    status,
    orgExists,
    projectExists,
    summary,
    actionLabel,
    actionURL,
    localCommunityEdition,
  };
}
