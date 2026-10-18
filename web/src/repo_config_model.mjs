import { PIPELINE_TEMPLATES } from "./pipeline_model.mjs";

// TRACEABILITY_MAPPING: REQ-056
export function validateRepoConfig(config) {
  const errors = [];
  if (!config || typeof config !== "object") {
    return { valid: false, errors: ["Configuration must be an object"] };
  }

  if (config.base_branch !== undefined && config.base_branch !== null) {
    const b = String(config.base_branch).trim();
    if (b.length > 0) {
      if (/\s/.test(b) || b.includes("..") || b.startsWith("/") || b.endsWith("/")) {
        errors.push("Base branch contains invalid characters");
      }
    }
  }

  if (config.copilot_handle !== undefined && config.copilot_handle !== null) {
    const h = String(config.copilot_handle).trim();
    if (h.length > 0) {
      if (!h.startsWith("@") || /\s/.test(h)) {
        errors.push("Agent handle must begin with '@' and contain no whitespace");
      }
    }
  }

  if (config.branch_prefix !== undefined && config.branch_prefix !== null) {
    const p = String(config.branch_prefix).trim();
    if (p.length > 0 && /\s/.test(p)) {
      errors.push("Branch prefix must not contain whitespace");
    }
  }

  return {
    valid: errors.length === 0,
    errors,
  };
}

// TRACEABILITY_MAPPING: REQ-056
export function normalizeRepoConfig(config) {
  const c = config || {};
  const defaultBranch = c.default_branch && c.default_branch.trim() ? c.default_branch.trim() : "main";
  const baseBranch = c.base_branch && c.base_branch.trim() ? c.base_branch.trim() : defaultBranch;
  const branchPrefix = c.branch_prefix && c.branch_prefix.trim() ? c.branch_prefix.trim() : "copilot/";
  const copilotHandle = c.copilot_handle && c.copilot_handle.trim() ? c.copilot_handle.trim() : "@copilot";
  const issueLabels = c.issue_labels && c.issue_labels.trim() ? c.issue_labels.trim() : "copilot, remediation";
  const copilotPrompt = c.copilot_prompt !== undefined ? String(c.copilot_prompt).trim() : "";

  return {
    default_branch: defaultBranch,
    base_branch: baseBranch,
    branch_prefix: branchPrefix,
    copilot_handle: copilotHandle,
    issue_labels: issueLabels,
    copilot_prompt: copilotPrompt,
  };
}

// TRACEABILITY_MAPPING: REQ-056
export function composeAgentPrompt(repo, options) {
  const r = repo || {};
  const opts = options || {};
  const norm = normalizeRepoConfig(r);

  const title = opts.title || "Remediate open-source standards";
  const details = opts.details || "";
  const taskSpec = opts.specificationYaml ? `\n\`\`\`yaml\n${opts.specificationYaml.trim()}\n\`\`\`\n` : "";

  let prompt = `${norm.copilot_handle} please address this task for ${r.full_name || r.name || "this repository"}.\n\n`;
  prompt += `### Objective: ${title}\n\n`;

  if (details) {
    prompt += `${details}\n\n`;
  }

  if (taskSpec) {
    prompt += `### Target Specification:\n${taskSpec}\n`;
  }

  prompt += `### Repository Directives & Constraints:\n`;
  prompt += `- **Target Base Branch**: Open any pull request against \`${norm.base_branch}\`.\n`;
  prompt += `- **Working Branch Prefix**: Name your working branch with \`${norm.branch_prefix}\` (e.g. \`${norm.branch_prefix}remediate-*\`).\n`;
  prompt += `- **Non-Destructive Execution**: Inspect existing configuration files before making changes. Never overwrite working CI workflows, test commands, or scripts.\n`;

  if (norm.copilot_prompt) {
    prompt += `- **Custom Repository Rules**: ${norm.copilot_prompt}\n`;
  }

  prompt += `\nPlease inspect the repository, implement the necessary changes, run existing test suites to verify, and open a pull request targeting \`${norm.base_branch}\`.`;

  return prompt;
}

// TRACEABILITY_MAPPING: REQ-056
export function copilotWorkflowPresets() {
  // PIPELINE_TEMPLATES stays the single source of each workflow's id, name and filename
  const preset = ({ id, name, filename }) => ({ id, name, filename });
  return {
    ci_failure_handler: preset(PIPELINE_TEMPLATES.copilot_ci_failure_handler),
    review_handler: preset(PIPELINE_TEMPLATES.copilot_review_handler),
  };
}
