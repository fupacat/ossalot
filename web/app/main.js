    import { summarizePortfolio, getStatusBadge, buildRemediationRequest } from '/src/model.mjs';
    import { buildTrafficSeries, aggregateReferrers } from '/src/metrics_model.mjs';
    import { filterOpportunities, formatPRPlanSummary } from '/src/discovery_model.mjs';
    import { groupInboxByStatus } from '/src/inbox_model.mjs';
    import { evaluateSetupCapabilities, computeSetupProgress } from '/src/setup_model.mjs';
    import { PIPELINE_TEMPLATES, generatePipelineWorkflow } from '/src/pipeline_model.mjs';
    import { generateHygieneConfigs } from '/src/hygiene_model.mjs';
    import { evaluateSonarOnboarding } from '/src/sonar_model.mjs';
    import { validateRepoConfig, normalizeRepoConfig, composeAgentPrompt, checkCopilotWorkflows } from '/src/repo_config_model.mjs';

    let state = {
      repos: [],
      selectedRepoId: '',
      metrics: [],
      profile: null,
      opportunities: [],
      jobs: [],
      settings: {},
      inboxStatusFilter: 'new',
      selectedPipelineType: 'sonar_analysis',
      sonarOnboarding: null,
    };

    async function api(path, options = {}) {
      const res = await fetch(path, {
        headers: { 'Content-Type': 'application/json' },
        ...options,
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        throw new Error(data.error || `HTTP ${res.status}`);
      }
      return data;
    }

    async function loadAll() {
      const repos = await api('/api/v1/repos').catch(() => []);
      state.repos = Array.isArray(repos) ? repos : [];
      state.settings = (await api('/api/v1/settings').catch(() => ({}))) || {};
      const jobs = await api('/api/v1/jobs').catch(() => []);
      state.jobs = Array.isArray(jobs) ? jobs : [];
      const opps = await api('/api/v1/opportunities').catch(() => []);
      state.opportunities = Array.isArray(opps) ? opps : [];
      if (state.selectedRepoId && !state.repos.some(r => r.id === state.selectedRepoId)) {
        state.selectedRepoId = '';
      }
      if (!state.selectedRepoId && state.repos.length > 0) {
        state.selectedRepoId = state.repos[0].id;
      }
      if (state.selectedRepoId) {
        const metrics = await api(`/api/v1/repos/${state.selectedRepoId}/metrics`).catch(() => []);
        state.metrics = Array.isArray(metrics) ? metrics : [];
        const profRes = await fetch(`/api/v1/repos/${state.selectedRepoId}/profile`);
        state.profile = profRes.ok ? await profRes.json() : null;
      } else {
        state.metrics = [];
        state.profile = null;
      }
      render();
    }

    function currentRepo() {
      return state.repos.find(r => r.id === state.selectedRepoId) || state.repos[0] || null;
    }

    function render() {
      const summary = summarizePortfolio(state.repos);
      const repo = currentRepo();
      const repoOppsAll = repo ? state.opportunities.filter(o => o.repo_id === repo.id) : state.opportunities;

      // Selector
      const sel = document.getElementById('repoSelector');
      sel.innerHTML = state.repos.map(r =>
        `<option value="${r.id}" ${r.id === (repo && repo.id) ? 'selected' : ''}>${r.full_name} (${r.standards_score}%)</option>`
      ).join('');

      // KPI Strip
      const latestSnap = state.metrics.length > 0 ? state.metrics[state.metrics.length - 1] : null;
      document.getElementById('kpiStrip').innerHTML = `
        <div class="bg-slate-900/90 border border-slate-800 rounded-xl p-3">
          <div class="text-[11px] text-slate-400">Tracked Repositories</div>
          <div class="text-xl font-bold text-white mt-0.5">${summary.totalRepos} <span class="text-xs font-normal text-emerald-400">(${summary.ownedRepos} owned)</span></div>
        </div>
        <div class="bg-slate-900/90 border border-slate-800 rounded-xl p-3">
          <div class="text-[11px] text-slate-400">Avg Standards Score</div>
          <div class="text-xl font-bold text-indigo-400 mt-0.5">${summary.avgStandardsScore}<span class="text-xs text-slate-400">/100</span></div>
        </div>
        <div class="bg-slate-900/90 border border-slate-800 rounded-xl p-3">
          <div class="text-[11px] text-slate-400">14d Unique Views / Clones</div>
          <div class="text-xl font-bold text-emerald-400 mt-0.5">${latestSnap ? latestSnap.views_unique : 0} <span class="text-xs text-slate-400">/ ${latestSnap ? latestSnap.clones_unique : 0} clones</span></div>
        </div>
        <div class="bg-slate-900/90 border border-slate-800 rounded-xl p-3">
          <div class="text-[11px] text-slate-400">Action Inbox Matches</div>
          <div class="text-xl font-bold text-amber-400 mt-0.5">${repoOppsAll.length} <span class="text-xs text-slate-400">(${state.opportunities.length} total)</span></div>
        </div>
      `;

      // Always render settings & jobs even when portfolio is empty
      document.getElementById('setGhToken').value = state.settings.github_token || '';
      document.getElementById('setLlmProvider').value = state.settings.llm_provider || 'gemini';
      document.getElementById('setLlmKey').value = state.settings.llm_api_key || '';
      document.getElementById('setSlackUrl').value = state.settings.slack_webhook_url || '';

      document.getElementById('jobsList').innerHTML = state.jobs.map(j => `
        <div class="flex items-center justify-between bg-slate-950 border border-slate-800 rounded-xl px-3 py-2">
          <div>
            <div class="font-medium text-slate-200">${j.job_name}</div>
            <div class="text-[11px] text-slate-500">${j.executed_at}</div>
          </div>
          <span class="px-2 py-0.5 rounded text-[10px] font-bold uppercase ${j.status === 'completed' ? 'bg-emerald-500/15 text-emerald-400' : 'bg-rose-500/15 text-rose-400'}">${j.status}</span>
        </div>
      `).join('') || `<p class="text-slate-400">No jobs executed yet.</p>`;

      if (!repo) {
        document.getElementById('repoHeroCard').innerHTML = `
          <div class="text-center py-6 space-y-3">
            <div class="text-sm font-semibold text-white">No Repositories Added Yet</div>
            <p class="text-xs text-slate-400 max-w-md mx-auto">Tap <b>+ Repo</b> in the top-right corner to inspect a live GitHub repository (such as <code class="text-indigo-300">fupacat/ossalot</code>) and run its 15-check standards audit.</p>
            <div class="pt-1 flex justify-center gap-2">
              <button onclick="openAddRepoModal()" class="text-xs px-3.5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-semibold">+ Add GitHub Repository</button>
            </div>
          </div>
        `;
        document.getElementById('checksList').innerHTML = `<p class="text-xs text-slate-400 py-3">Add a repository to view its 15-check audit results.</p>`;
        document.getElementById('pillarBars').innerHTML = `<p class="text-xs text-slate-400">No repository selected.</p>`;
        document.getElementById('portfolioCountBadge').textContent = `0 repos`;
        document.getElementById('portfolioList').innerHTML = `<p class="text-xs text-slate-400">Empty portfolio.</p>`;
        return;
      }

      // Repo Hero
      document.getElementById('repoHeroCard').innerHTML = `
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div class="space-y-1">
            <div class="flex items-center gap-2 flex-wrap">
              <a href="https://github.com/${repo.full_name}" target="_blank" rel="noopener" class="text-base sm:text-lg font-bold text-white hover:text-indigo-300 underline decoration-slate-700">${repo.full_name} &nearr;</a>
              <span class="text-[11px] px-2 py-0.5 rounded-md bg-slate-800 text-slate-300">${repo.primary_language || 'Polyglot'}</span>
              <span class="text-[11px] px-2 py-0.5 rounded-md ${repo.is_owned ? 'bg-indigo-500/20 text-indigo-300 border border-indigo-500/30' : 'bg-slate-800 text-slate-400'}">${repo.is_owned ? 'Owned Repo (1-Click PR Enabled)' : 'External Public Repo'}</span>
              <button onclick="deleteRepo('${repo.id}')" class="text-[11px] px-2 py-0.5 rounded-md bg-rose-500/10 hover:bg-rose-500/20 text-rose-300 border border-rose-500/20">Remove</button>
            </div>
            <p class="text-xs text-slate-300">${repo.description || 'No description provided'}</p>
            <div class="flex gap-1.5 flex-wrap pt-1">
              ${(repo.topics || []).map(t => `<span class="text-[10px] px-2 py-0.5 rounded-full bg-slate-800 text-slate-400">#${t}</span>`).join('')}
            </div>
          </div>
          <div class="flex items-center gap-3 bg-slate-950 border border-slate-800 rounded-xl px-3.5 py-2">
            <div class="text-right">
              <div class="text-[10px] uppercase tracking-wider text-slate-400">Standards Score</div>
              <div class="text-2xl font-extrabold ${repo.standards_score >= 80 ? 'text-emerald-400' : 'text-amber-400'}">${repo.standards_score}<span class="text-xs text-slate-500">/100</span></div>
            </div>
          </div>
        </div>
      `;

      // Checks List
      const checks = (repo.latest_audit && repo.latest_audit.checks) || [];
      document.getElementById('checksList').innerHTML = checks.map(c => {
        const badge = getStatusBadge(c.status);
        const badgeColors = badge.tone === 'emerald'
          ? 'bg-emerald-500/15 text-emerald-400 border-emerald-500/30'
          : badge.tone === 'amber'
          ? 'bg-amber-500/15 text-amber-400 border-amber-500/30'
          : 'bg-rose-500/15 text-rose-400 border-rose-500/30';
        return `
          <div class="py-2.5 flex items-center justify-between gap-2">
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <span class="text-[10px] font-bold px-2 py-0.5 rounded border ${badgeColors}">${badge.label}</span>
                <span class="text-xs font-medium text-slate-200 truncate">${c.check_key}</span>
                <span class="text-[11px] text-slate-500">(${c.score}/${c.weight} pts)</span>
              </div>
              <p class="text-xs text-slate-400 mt-0.5">${c.details}</p>
            </div>
            ${c.status !== 'pass' ? `
              <button onclick="openRemediate('${c.check_key}')" class="shrink-0 text-xs px-2.5 py-1 rounded-lg bg-indigo-600/20 hover:bg-indigo-600/40 text-indigo-300 border border-indigo-500/30 font-medium">
                Remediate
              </button>
            ` : `<span class="text-[11px] text-emerald-400 shrink-0">&check; Verified</span>`}
          </div>
        `;
      }).join('') || `<p class="text-xs text-slate-400 py-3">No audit checks run yet. Tap "Re-Audit" above.</p>`;

      // 4-Pillar Bars
      const cats = (repo.latest_audit && repo.latest_audit.category_scores) || {};
      const pillarNames = [
        ['community_files', 'Community & Governance'],
        ['documentation_quality', 'Documentation & Onboarding'],
        ['technical_hygiene', 'Technical & Release Hygiene'],
        ['security_openssf', 'Security & OpenSSF Scorecard'],
      ];
      document.getElementById('pillarBars').innerHTML = pillarNames.map(([key, label]) => {
        const val = cats[key] || 0;
        const pct = Math.round((val / 25) * 100);
        return `
          <div class="space-y-1">
            <div class="flex justify-between text-xs">
              <span class="text-slate-300">${label}</span>
              <span class="font-semibold text-white">${val}/25</span>
            </div>
            <div class="h-2 rounded-full bg-slate-800 overflow-hidden">
              <div class="h-full bg-gradient-to-r from-indigo-500 to-emerald-400" style="width:${pct}%"></div>
            </div>
          </div>
        `;
      }).join('');

      // Portfolio sidebar list
      document.getElementById('portfolioCountBadge').textContent = `${state.repos.length} repos`;
      document.getElementById('portfolioList').innerHTML = state.repos.map(r => `
        <button onclick="selectRepo('${r.id}')" class="w-full text-left p-2.5 rounded-xl border transition flex items-center justify-between ${r.id === repo.id ? 'bg-slate-800/90 border-indigo-500/50' : 'bg-slate-950/60 border-slate-800 hover:bg-slate-800/40'}">
          <div class="truncate pr-2">
            <div class="text-xs font-semibold text-white truncate">${r.full_name}</div>
            <div class="text-[11px] text-slate-400">${r.primary_language || 'Repo'} &bull; ${r.is_owned ? 'Owned' : 'Public'}</div>
          </div>
          <span class="text-xs font-bold px-2 py-0.5 rounded bg-slate-900 text-emerald-400">${r.standards_score}%</span>
        </button>
      `).join('');

      // Traffic & Registries Tab
      const series = buildTrafficSeries(state.metrics);
      const refs = aggregateReferrers(state.metrics);
      document.getElementById('trafficChartContainer').innerHTML = series.length === 0
        ? `<p class="text-xs text-slate-400">No snapshots yet. Tap "+ Snapshot Today" to archive GitHub traffic.</p>`
        : `
          <div class="grid grid-cols-1 sm:grid-cols-3 gap-2.5">
            ${series.map(pt => `
              <div class="bg-slate-950 border border-slate-800 rounded-xl p-3 space-y-1">
                <div class="flex items-center justify-between">
                  <span class="text-xs font-semibold text-indigo-300">${pt.date}</span>
                  <span class="text-[11px] px-1.5 py-0.5 rounded bg-emerald-500/10 text-emerald-400">Score ${pt.adoptionScore}</span>
                </div>
                <div class="text-xs text-slate-300">Stars: <b>${pt.stars}</b> &bull; Unique Views: <b>${pt.viewsUnique}</b></div>
                <div class="text-xs text-slate-400">Unique Clones: <b>${pt.clonesUnique}</b> &bull; Pkg Downloads: <b>${pt.registryDownloads}</b></div>
              </div>
            `).join('')}
          </div>
        `;

      document.getElementById('referrersTable').innerHTML = refs.length === 0
        ? `<p class="text-xs text-slate-400">No referrer data recorded yet.</p>`
        : refs.map(r => `
          <div class="flex items-center justify-between bg-slate-950 border border-slate-800 rounded-xl px-3 py-2 text-xs">
            <span class="font-medium text-slate-200">${r.referrer}</span>
            <span class="text-emerald-400 font-semibold">${r.count} views <span class="text-slate-500">(${r.uniques} unique)</span></span>
          </div>
        `).join('');

      const dlMap = (latestSnap && latestSnap.registry_downloads) || {};
      document.getElementById('registryCards').innerHTML = Object.entries(dlMap).map(([eco, count]) => `
        <div class="bg-slate-950 border border-slate-800 rounded-xl p-3">
          <div class="text-[10px] uppercase tracking-wider text-slate-400">${eco}</div>
          <div class="text-lg font-bold text-white mt-0.5">${Number(count).toLocaleString()}</div>
          <div class="text-[10px] text-emerald-400">downloads/day</div>
        </div>
      `).join('') || `<p class="text-xs text-slate-400 col-span-2">No registry coordinates configured.</p>`;

      // Opportunities Tab
      if (state.profile) {
        document.getElementById('profileContainer').innerHTML = `
          <div class="bg-slate-950 border border-slate-800 rounded-xl p-3 space-y-2">
            <div class="text-slate-200 font-medium">${state.profile.summary}</div>
            <div class="flex flex-wrap gap-1.5">
              ${(state.profile.ecosystem_keywords || []).map(k => `<span class="px-2 py-0.5 rounded bg-indigo-500/20 text-indigo-300 text-[11px]">${k}</span>`).join('')}
            </div>
            <div class="text-slate-400"><b>Search Queries:</b> ${(state.profile.search_queries || []).map(q => `<code class="bg-slate-900 px-1.5 py-0.5 rounded text-emerald-300">${q}</code>`).join(' ')}</div>
          </div>
        `;
      } else {
        document.getElementById('profileContainer').innerHTML = `<p class="text-xs text-slate-400">Tap "Scan GitHub Issues" to build the AI Capability Profile for ${repo.full_name}.</p>`;
      }

      const repoOpps = state.opportunities.filter(o => o.repo_id === repo.id);
      const ghOpps = filterOpportunities(repoOpps, { category: 'github_issue', minScore: 70 });
      document.getElementById('githubOpportunitiesList').innerHTML = ghOpps.map(o => {
        const plan = formatPRPlanSummary(o);
        return `
          <div class="bg-slate-950 border border-slate-800 rounded-xl p-3.5 space-y-2">
            <div class="flex items-start justify-between gap-2">
              <div>
                <span class="text-[10px] font-bold px-2 py-0.5 rounded bg-emerald-500/20 text-emerald-400">${o.relevance_score}% Match</span>
                <a href="${o.url}" target="_blank" rel="noopener" class="text-xs text-indigo-300 font-medium ml-1 hover:underline">${o.source_community} &nearr;</a>
                <h4 class="text-sm font-semibold text-white mt-1">${o.title}</h4>
              </div>
              <button onclick="showOpportunityDraft('${o.id}')" class="shrink-0 text-xs px-2.5 py-1 rounded-lg bg-indigo-600 text-white font-medium">View PR Plan</button>
            </div>
            <p class="text-xs text-slate-400">${o.legitimacy_reason}</p>
            <div class="text-[11px] text-slate-500">${plan.stepsCount} implementation steps ready &bull; Status: <b class="text-slate-300">${o.status}</b></div>
          </div>
        `;
      }).join('') || `<p class="text-xs text-slate-400">Tap "Scan GitHub Issues" to discover external projects needing ${repo.name}.</p>`;

      // Community & Playbooks Tab
      const commOpps = repoOpps.filter(o => o.category !== 'github_issue' && o.category !== 'marketing_playbook');
      const playbooks = repoOpps.filter(o => o.category === 'marketing_playbook');

      document.getElementById('communityThreadsList').innerHTML = commOpps.map(o => `
        <div class="bg-slate-950 border border-slate-800 rounded-xl p-3 space-y-1.5">
          <div class="flex items-center justify-between">
            <span class="text-[10px] font-semibold uppercase px-2 py-0.5 rounded bg-indigo-500/20 text-indigo-300">${o.category} &bull; ${o.relevance_score}%</span>
            <button onclick="showOpportunityDraft('${o.id}')" class="text-xs text-emerald-400 hover:underline">Draft Reply &rarr;</button>
          </div>
          <a href="${o.url}" target="_blank" rel="noopener" class="block text-xs font-semibold text-white hover:text-indigo-300 underline decoration-slate-700">${o.title} &nearr;</a>
          <p class="text-[11px] text-slate-400">${o.legitimacy_reason}</p>
        </div>
      `).join('') || `<p class="text-xs text-slate-400">Tap "Scan Feeds" to scan HN, Reddit, StackOverflow, Dev.to & Lobsters.</p>`;

      document.getElementById('playbooksList').innerHTML = playbooks.map(o => `
        <div class="bg-slate-950 border border-slate-800 rounded-xl p-3 space-y-1.5">
          <div class="flex items-center justify-between">
            <span class="text-[10px] font-semibold px-2 py-0.5 rounded bg-emerald-500/15 text-emerald-400">${o.source_community}</span>
            <button onclick="showOpportunityDraft('${o.id}')" class="text-xs px-2 py-0.5 rounded bg-slate-800 text-indigo-300">Open Draft</button>
          </div>
          <div class="text-xs font-semibold text-white">${o.draft_title}</div>
          <p class="text-[11px] text-amber-300/90"><b>Guardrail:</b> ${o.community_rules_notes}</p>
        </div>
      `).join('') || `<p class="text-xs text-slate-400">Tap "Generate Playbooks" to build launch posts.</p>`;

      // Unified Action Inbox Tab
      const buckets = groupInboxByStatus(repoOpps);
      const statuses = ['new', 'saved', 'drafted', 'submitted', 'dismissed'];
      document.getElementById('inboxStatusTabs').innerHTML = statuses.map(st => `
        <button onclick="setInboxFilter('${st}')" class="text-xs px-2.5 py-1 rounded-lg font-medium capitalize ${state.inboxStatusFilter === st ? 'bg-indigo-600 text-white' : 'bg-slate-950 text-slate-400 border border-slate-800'}">
          ${st} (${(buckets[st] || []).length})
        </button>
      `).join('');

      const activeInbox = buckets[state.inboxStatusFilter] || [];
      document.getElementById('inboxItemsList').innerHTML = activeInbox.map(o => `
        <div class="bg-slate-950 border border-slate-800 rounded-xl p-3.5 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          <div class="space-y-1">
            <div class="flex items-center gap-2">
              <span class="text-[10px] font-bold uppercase px-2 py-0.5 rounded bg-slate-800 text-indigo-300">${o.category}</span>
              <span class="text-xs font-semibold text-emerald-400">${o.relevance_score}% Fit</span>
              <a href="${o.url}" target="_blank" rel="noopener" class="text-xs text-slate-400 hover:underline">${o.source_community} &nearr;</a>
            </div>
            <div class="text-sm font-semibold text-white">${o.title}</div>
            <p class="text-xs text-slate-400">${o.legitimacy_reason}</p>
          </div>
          <div class="flex items-center gap-1.5 shrink-0 flex-wrap">
            <button onclick="showOpportunityDraft('${o.id}')" class="text-xs px-2.5 py-1 rounded-lg bg-slate-800 text-white">Draft / Plan</button>
            <button onclick="updateOppStatus('${o.id}', 'saved')" class="text-xs px-2.5 py-1 rounded-lg bg-indigo-600/20 text-indigo-300 border border-indigo-500/30">Save</button>
            <button onclick="updateOppStatus('${o.id}', 'submitted')" class="text-xs px-2.5 py-1 rounded-lg bg-emerald-600/20 text-emerald-300 border border-emerald-500/30">Mark Submitted</button>
            <button onclick="updateOppStatus('${o.id}', 'dismissed')" class="text-xs px-2 py-1 rounded-lg bg-rose-600/15 text-rose-300">Dismiss</button>
          </div>
        </div>
      `).join('') || `<p class="text-xs text-slate-400">No items in "${state.inboxStatusFilter}" status for ${repo ? repo.full_name : 'portfolio'}.</p>`;

      // Render Setup & Capabilities (REQ-034)
      const caps = evaluateSetupCapabilities(state.settings);
      const setupProgress = computeSetupProgress(caps);
      document.getElementById('setupProgressBar').innerHTML = `
        <span class="text-xs text-slate-400 font-medium">${setupProgress.completed}/${setupProgress.total} Unlocked</span>
        <div class="w-24 bg-slate-800 rounded-full h-2">
          <div class="bg-indigo-500 h-2 rounded-full" style="width: ${setupProgress.percentage}%"></div>
        </div>
      `;

      const capKeys = Object.keys(caps);
      document.getElementById('setupCapabilityCards').innerHTML = capKeys.map(k => {
        const c = caps[k];
        const isEnabled = c.status === 'enabled';
        const tone = isEnabled ? 'bg-emerald-500/10 border-emerald-500/30 text-emerald-400' : 'bg-slate-900 border-slate-800 text-amber-400';
        return `
          <div class="p-3.5 rounded-xl border ${tone} space-y-1.5 bg-slate-900/80">
            <div class="flex items-center justify-between">
              <span class="font-semibold text-white capitalize">${k.replace(/_/g, ' ')}</span>
              <span class="text-[10px] uppercase font-bold px-2 py-0.5 rounded ${isEnabled ? 'bg-emerald-500/20 text-emerald-300' : 'bg-amber-500/20 text-amber-300'}">${c.status}</span>
            </div>
            <p class="text-[11px] text-slate-300">${c.details}</p>
            <div class="text-[10px] text-slate-400">Required: <code>${c.scope_hint}</code></div>
          </div>
        `;
      }).join('');

      document.getElementById('setupGhToken').value = state.settings.github_token || '';
      document.getElementById('setupSonarToken').value = state.settings.sonar_token || '';
      document.getElementById('setupSonarHost').value = state.settings.sonar_host_url || '';
      document.getElementById('setupSonarOrg').value = state.settings.sonar_org || '';
      document.getElementById('setupLlmProvider').value = state.settings.llm_provider || 'gemini';
      document.getElementById('setupLlmKey').value = state.settings.llm_api_key || '';
      document.getElementById('setupSlackUrl').value = state.settings.slack_webhook_url || '';

      // Render Sonar Onboarding (REQ-037)
      const sonarEl = document.getElementById('sonarOnboardingResults');
      if (state.sonarOnboarding) {
        const so = state.sonarOnboarding;
        const isReady = so.status === 'registered_and_imported';
        const isOrgOnly = so.status === 'registered_not_imported';
        sonarEl.innerHTML = `
          <div class="p-3.5 rounded-xl border ${isReady ? 'bg-emerald-500/10 border-emerald-500/30' : isOrgOnly ? 'bg-indigo-500/10 border-indigo-500/30' : 'bg-amber-500/10 border-amber-500/30'} space-y-2.5">
            <div class="flex items-center justify-between">
              <span class="font-bold text-white text-xs">${so.repo} &bull; ${so.projectKey}</span>
              <span class="text-[10px] font-bold uppercase px-2 py-0.5 rounded ${isReady ? 'bg-emerald-500/20 text-emerald-300' : 'bg-indigo-500/20 text-indigo-300'}">${so.status.replace(/_/g, ' ')}</span>
            </div>
            <p class="text-slate-300">${so.summary}</p>
            <div class="flex items-center gap-2 pt-1 flex-wrap">
              <a href="${so.actionURL}" target="_blank" rel="noopener" class="text-xs px-3 py-1.5 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white font-semibold inline-flex items-center gap-1">${so.actionLabel} &nearr;</a>
              <a href="${so.tokenURL}" target="_blank" rel="noopener" class="text-xs px-3 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 font-semibold inline-flex items-center gap-1">Create SonarCloud token &nearr;</a>
              <a href="${so.secretURL}" target="_blank" rel="noopener" class="text-xs px-3 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 font-semibold inline-flex items-center gap-1">Add ${so.secretName} secret &nearr;</a>
              ${so.projectExists ? `<img src="https://sonarcloud.io/api/project_badges/measure?project=${so.projectKey}&metric=alert_status" alt="Sonar Quality Gate" class="h-6" />` : ''}
            </div>
          </div>

          <!-- Local Community Edition Box -->
          <div class="bg-slate-900/90 border border-slate-800 rounded-xl p-3.5 space-y-2">
            <div class="flex items-center justify-between">
              <span class="font-semibold text-slate-200">${so.localCommunityEdition.title}</span>
              <span class="text-[10px] px-2 py-0.5 rounded bg-slate-800 text-slate-400">Zero Cloud Setup</span>
            </div>
            <p class="text-[11px] text-slate-400">${so.localCommunityEdition.description}</p>
            <div class="space-y-1">
              <span class="text-[10px] text-slate-400 uppercase font-semibold">1. Start Local Container:</span>
              <pre class="bg-slate-950 p-2 rounded text-[11px] text-emerald-300 font-mono overflow-x-auto whitespace-pre">${so.localCommunityEdition.dockerCommand}</pre>
            </div>
            <div class="space-y-1">
              <span class="text-[10px] text-slate-400 uppercase font-semibold">2. Scan from Terminal:</span>
              <pre class="bg-slate-950 p-2 rounded text-[11px] text-indigo-300 font-mono overflow-x-auto whitespace-pre">${so.localCommunityEdition.localScanHint}</pre>
            </div>
          </div>
        `;
      } else {
        sonarEl.innerHTML = `<p class="text-slate-400">Select a repository above and tap "Check Sonar Status for Current Repo" to run live detection.</p>`;
      }

      // Render Pipelines (REQ-035)
      const pipelineTypes = Object.keys(PIPELINE_TEMPLATES);
      document.getElementById('pipelineTemplatesList').innerHTML = pipelineTypes.map(t => {
        const item = PIPELINE_TEMPLATES[t];
        const isSel = state.selectedPipelineType === t;
        return `
          <div onclick="selectPipelinePreset('${t}')" class="p-4 rounded-xl border cursor-pointer transition ${isSel ? 'border-indigo-500 bg-indigo-950/20 shadow-md' : 'border-slate-800 bg-slate-950/60 hover:border-slate-700'}">
            <div class="flex items-center justify-between mb-1.5">
              <span class="text-xs font-semibold text-white">${item.name}</span>
              <span class="text-[10px] text-indigo-400 font-mono">${item.filename}</span>
            </div>
            <p class="text-xs text-slate-400">${item.description}</p>
          </div>
        `;
      }).join('');

      const activePipe = generatePipelineWorkflow(state.selectedPipelineType, repo ? repo.full_name : 'owner/repo');
      document.getElementById('pipelinePreviewContainer').classList.remove('hidden');
      document.getElementById('pipelinePreviewTitle').textContent = PIPELINE_TEMPLATES[state.selectedPipelineType].name;
      document.getElementById('pipelinePreviewFilename').textContent = activePipe.filename;
      document.getElementById('pipelinePreviewCode').textContent = activePipe.content;

      // Render Code Hygiene (REQ-036)
      const repoLang = repo ? repo.primary_language : 'Go';
      const hygiene = generateHygieneConfigs(repoLang, repo ? repo.full_name : 'project');
      const hygieneKeys = Object.keys(hygiene);
      document.getElementById('hygieneCardsContainer').innerHTML = hygieneKeys.map(hk => {
        const h = hygiene[hk];
        if (!h) return '';
        return `
          <div class="bg-slate-950 border border-slate-800 rounded-xl p-4 space-y-2">
            <div class="flex items-center justify-between">
              <div>
                <span class="text-xs font-semibold text-white">${h.filename}</span>
                <p class="text-[11px] text-slate-400">${h.description}</p>
              </div>
              <button onclick="navigator.clipboard.writeText(decodeURIComponent('${encodeURIComponent(h.content)}'))" class="text-xs px-2.5 py-1 rounded bg-slate-800 hover:bg-slate-700 text-indigo-300">Copy</button>
            </div>
            <pre class="bg-slate-900 border border-slate-800 rounded-lg p-2.5 text-[11px] text-emerald-300 font-mono overflow-x-auto max-h-40 whitespace-pre">${h.content}</pre>
          </div>
        `;
      }).join('');

      // Render Repository, Branch & Copilot Configuration (REQ-056)
      if (repo) {
        const norm = normalizeRepoConfig(repo);
        const ownedBadge = document.getElementById('repoConfigOwnedBadge');
        if (ownedBadge) {
          if (repo.is_owned) {
            ownedBadge.className = 'text-xs px-2.5 py-1 rounded-full bg-emerald-500/15 text-emerald-400 border border-emerald-500/30 font-medium';
            ownedBadge.textContent = 'Owned Workspace Repo';
          } else {
            ownedBadge.className = 'text-xs px-2.5 py-1 rounded-full bg-amber-500/15 text-amber-400 border border-amber-500/30 font-medium';
            ownedBadge.textContent = 'External / Public Repo';
          }
        }
        const branchBadge = document.getElementById('repoConfigBranchBadge');
        if (branchBadge) {
          branchBadge.textContent = `Target: ${norm.base_branch}`;
        }
        const ghLink = document.getElementById('repoConfigGhLink');
        if (ghLink) {
          ghLink.href = repo.html_url || `https://github.com/${repo.full_name}`;
        }

        const sel = document.getElementById('repoConfigRepoSelect');
        if (sel) {
          sel.innerHTML = state.repos.map(r =>
            `<option value="${r.id}" ${r.id === repo.id ? 'selected' : ''}>${r.full_name}</option>`
          ).join('');
        }

        const descInput = document.getElementById('repoConfigDesc');
        if (descInput) descInput.value = repo.description || '';
        const isOwnedBox = document.getElementById('repoConfigIsOwned');
        if (isOwnedBox) isOwnedBox.checked = Boolean(repo.is_owned);
        const defBranchInput = document.getElementById('repoConfigDefBranch');
        if (defBranchInput) defBranchInput.value = repo.default_branch || norm.default_branch;
        const baseBranchInput = document.getElementById('repoConfigBaseBranch');
        if (baseBranchInput) baseBranchInput.value = repo.base_branch || norm.base_branch;
        const branchPrefixInput = document.getElementById('repoConfigBranchPrefix');
        if (branchPrefixInput) branchPrefixInput.value = repo.branch_prefix || norm.branch_prefix;
        const agentHandleInput = document.getElementById('repoConfigAgentHandle');
        if (agentHandleInput) agentHandleInput.value = repo.copilot_handle || norm.copilot_handle;
        const issueLabelsInput = document.getElementById('repoConfigIssueLabels');
        if (issueLabelsInput) issueLabelsInput.value = repo.issue_labels || norm.issue_labels;
        const promptInput = document.getElementById('repoConfigPrompt');
        if (promptInput) promptInput.value = repo.copilot_prompt || '';

        const workflows = checkCopilotWorkflows({
          ".github/workflows/copilot-ci-failure-handler.yml": Boolean(repo.latest_audit && repo.latest_audit.checks && repo.latest_audit.checks.some(c => c.check_key === 'copilot_ci_failure_handler' && c.status === 'pass')),
          ".github/workflows/copilot-review-handler.yml": Boolean(repo.latest_audit && repo.latest_audit.checks && repo.latest_audit.checks.some(c => c.check_key === 'copilot_review_handler' && c.status === 'pass')),
        });

        const wfList = document.getElementById('copilotWorkflowsList');
        if (wfList) {
          wfList.innerHTML = Object.values(workflows).map(wf => `
            <div class="bg-slate-900 border border-slate-800 rounded-xl p-3 flex items-center justify-between">
              <div>
                <div class="text-xs font-semibold text-white">${wf.name}</div>
                <div class="text-[11px] text-indigo-300 font-mono">${wf.filename}</div>
              </div>
              <button type="button" onclick="dispatchWorkflowPreset('${wf.id}')" class="text-xs px-2.5 py-1 rounded-lg bg-indigo-600/20 hover:bg-indigo-600/40 text-indigo-300 border border-indigo-500/30">
                Dispatch to Agent
              </button>
            </div>
          `).join('');
        }

        window.updateRepoPromptPreview();
      }
    }

    // Expose handlers to window
    window.switchTab = (tab) => {
      document.querySelectorAll('[id^="tab-"]').forEach(el => el.classList.add('hidden'));
      document.getElementById(`tab-${tab}`).classList.remove('hidden');
      document.querySelectorAll('.tab-btn').forEach(btn => {
        if (btn.dataset.tab === tab) {
          btn.className = 'tab-btn whitespace-nowrap px-3 py-1.5 rounded-lg text-xs font-medium bg-indigo-600 text-white';
        } else {
          btn.className = 'tab-btn whitespace-nowrap px-3 py-1.5 rounded-lg text-xs font-medium bg-slate-900 text-slate-300 hover:bg-slate-800';
        }
      });
      if (tab === 'setup') {
        window.checkSonarOnboardingNow();
      }
    };

    window.selectRepo = async (id) => {
      state.selectedRepoId = id;
      await loadAll();
    };

    window.deleteRepo = async (id) => {
      await api(`/api/v1/repos/${id}`, { method: 'DELETE' });
      if (state.selectedRepoId === id) {
        state.selectedRepoId = null;
      }
      await loadAll();
    };

    window.openRemediate = async (checkKey) => {
      const repo = currentRepo();
      document.getElementById('modalTitle').textContent = `Loading Remediation for ${checkKey}...`;
      document.getElementById('modalSubtitle').textContent = 'Generating compliant content and branch coordinates...';
      document.getElementById('modalContent').textContent = 'Please wait...';
      document.getElementById('modalFooter').innerHTML = '';
      document.getElementById('modalOverlay').classList.remove('hidden');

      try {
        const payload = buildRemediationRequest(checkKey, false);
        const res = await api(`/api/v1/repos/${repo.id}/remediate`, {
          method: 'POST',
          body: JSON.stringify(payload),
        });
        const draft = res.draft || {};
        document.getElementById('modalTitle').textContent = `Remediation: ${draft.file_path || checkKey}`;
        document.getElementById('modalSubtitle').textContent = `Branch: ${draft.branch_name} • Commit: ${draft.commit_message}`;
        document.getElementById('modalContent').textContent = draft.content || JSON.stringify(res, null, 2);
        document.getElementById('modalFooter').innerHTML = `
          <button onclick="navigator.clipboard.writeText(document.getElementById('modalContent').textContent)" class="text-xs px-3 py-1.5 rounded-lg bg-slate-800 text-slate-200">Copy File</button>
          ${repo.is_owned ? `
            <button id="dispatchIssueBtn" onclick="dispatchAgentIssue('${checkKey}', '${draft.file_path || checkKey}')" class="text-xs px-3.5 py-1.5 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white font-medium">1-Click Dispatch Issue (@copilot)</button>
            <button id="createPRBtn" onclick="createRemediationPR('${checkKey}')" class="text-xs px-3 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 font-medium border border-slate-700">Direct PR</button>
          ` : ''}
        `;
      } catch (err) {
        document.getElementById('modalSubtitle').innerHTML = `<span class="text-rose-400 font-semibold">&cross; ${err.message}</span>`;
      }
    };

    window.dispatchAgentIssue = async (checkKey, filePath) => {
      const repo = currentRepo();
      const norm = normalizeRepoConfig(repo);
      const btn = document.getElementById('dispatchIssueBtn');
      if (btn) {
        btn.disabled = true;
        btn.textContent = `Filing issue on GitHub for ${norm.copilot_handle}...`;
      }
      const content = document.getElementById('modalContent').textContent;
      const title = `[OSSAlot] Configure ${checkKey} (${filePath || 'Standards Remediation'})`;
      const body = composeAgentPrompt(repo, {
        title: `Configure ${checkKey} (${filePath || 'Standards Remediation'})`,
        details: `OSSAlot standards audit identified an opportunity to improve repository health: **\`${checkKey}\`**.\nTarget File: \`${filePath || checkKey}\``,
        specificationYaml: content,
      });

      const labels = (repo.issue_labels || 'copilot, remediation')
        .split(',')
        .map(l => l.trim())
        .filter(Boolean);

      try {
        const res = await api(`/api/v1/repos/${repo.id}/issues`, {
          method: 'POST',
          body: JSON.stringify({
            title,
            body,
            labels,
          }),
        });
        if (res.html_url) {
          document.getElementById('modalSubtitle').innerHTML = `
            <div class="mt-2 p-2.5 rounded-xl bg-indigo-500/20 border border-indigo-500/30 text-indigo-300 font-medium text-xs flex flex-wrap items-center justify-between gap-2">
              <span>&check; GitHub Issue #${res.number} Filed for ${norm.copilot_handle}!</span>
              <a href="${res.html_url}" target="_blank" rel="noopener" class="underline font-bold text-white bg-indigo-600 hover:bg-indigo-500 px-3 py-1 rounded-lg">View Issue on GitHub &nearr;</a>
            </div>
          `;
          if (btn) {
            btn.outerHTML = `<a href="${res.html_url}" target="_blank" rel="noopener" class="text-xs px-3.5 py-1.5 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white font-semibold">Open Issue on GitHub &nearr;</a>`;
          }
        }
      } catch (err) {
        document.getElementById('modalSubtitle').innerHTML = `
          <div class="mt-2 p-2.5 rounded-xl bg-rose-500/20 border border-rose-500/30 text-rose-300 font-medium text-xs">
            &cross; Error filing GitHub Issue: ${err.message || 'Check repository permissions'}
          </div>
        `;
        if (btn) {
          btn.disabled = false;
          btn.textContent = `Retry Dispatch Issue (${norm.copilot_handle})`;
        }
      }
    };

    window.createRemediationPR = async (checkKey) => {
      const repo = currentRepo();
      const btn = document.getElementById('createPRBtn');
      if (btn) {
        btn.disabled = true;
        btn.textContent = 'Creating branch, commit & Pull Request on GitHub...';
      }
      try {
        const res = await api(`/api/v1/repos/${repo.id}/remediate`, {
          method: 'POST',
          body: JSON.stringify(buildRemediationRequest(checkKey, true)),
        });
        if (res.pr_url) {
          document.getElementById('modalSubtitle').innerHTML = `
            <div class="mt-2 p-2.5 rounded-xl bg-emerald-500/20 border border-emerald-500/30 text-emerald-300 font-medium text-xs flex flex-wrap items-center justify-between gap-2">
              <span>&check; Pull Request Created on GitHub!</span>
              <a href="${res.pr_url}" target="_blank" rel="noopener" class="underline font-bold text-white bg-emerald-600 hover:bg-emerald-500 px-3 py-1 rounded-lg">View PR on GitHub &nearr;</a>
            </div>
          `;
          if (btn) {
            btn.outerHTML = `<a href="${res.pr_url}" target="_blank" rel="noopener" class="text-xs px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-500 text-white font-semibold">Open PR on GitHub &nearr;</a>`;
          }
        } else {
          document.getElementById('modalSubtitle').innerHTML = `<span class="text-emerald-400 font-semibold">Pull Request Created Successfully</span>`;
        }
      } catch (err) {
        document.getElementById('modalSubtitle').innerHTML = `
          <div class="mt-2 p-2.5 rounded-xl bg-rose-500/20 border border-rose-500/30 text-rose-300 font-medium text-xs">
            &cross; Error creating GitHub PR: ${err.message || 'Check repository permissions'}
          </div>
        `;
        if (btn) {
          btn.disabled = false;
          btn.textContent = 'Retry 1-Click Open GitHub PR';
        }
      }
    };

    window.showOpportunityDraft = (id) => {
      const opp = state.opportunities.find(o => o.id === id);
      if (!opp) return;
      document.getElementById('modalTitle').textContent = opp.draft_title || opp.title;
      document.getElementById('modalSubtitle').textContent = `${opp.source_community} (${opp.relevance_score}% Relevance) • Action: ${opp.recommended_action}`;
      const bodyText = opp.pr_implementation_plan
        ? `=== PR IMPLEMENTATION PLAN ===\n${opp.pr_implementation_plan}\n\n=== DRAFT COMMENT / PR BODY ===\n${opp.draft_body}`
        : `=== COMMUNITY GUARDRAILS ===\n${opp.community_rules_notes || 'Lead with technical value.'}\n\n=== DRAFT POST ===\n${opp.draft_body}`;
      document.getElementById('modalContent').textContent = bodyText;
      document.getElementById('modalFooter').innerHTML = `
        <button onclick="navigator.clipboard.writeText(document.getElementById('modalContent').textContent)" class="text-xs px-3 py-1.5 rounded-lg bg-indigo-600 text-white font-medium">Copy Draft</button>
      `;
      document.getElementById('modalOverlay').classList.remove('hidden');
    };

    window.closeModal = () => document.getElementById('modalOverlay').classList.add('hidden');
    window.openAddRepoModal = () => {
      document.getElementById('addRepoError').classList.add('hidden');
      document.getElementById('addRepoModal').classList.remove('hidden');
    };
    window.closeAddRepoModal = () => document.getElementById('addRepoModal').classList.add('hidden');

    window.submitAddRepo = async (e) => {
      e.preventDefault();
      const errEl = document.getElementById('addRepoError');
      const btn = document.getElementById('addRepoSubmitBtn');
      errEl.classList.add('hidden');
      const url = document.getElementById('newRepoUrl').value.trim();
      const isOwned = document.getElementById('newRepoOwned').checked;
      btn.disabled = true;
      btn.textContent = '1/3 Inspecting & Auditing on GitHub...';
      try {
        const created = await api('/api/v1/repos', {
          method: 'POST',
          body: JSON.stringify({ url, is_owned: isOwned }),
        });
        if (created && created.id) {
          state.selectedRepoId = created.id;
          await api(`/api/v1/repos/${created.id}/audit`, { method: 'POST' }).catch(() => {});
          btn.textContent = '2/3 Collecting Traffic & Capability Profile...';
          await api(`/api/v1/repos/${created.id}/metrics/collect`, {
            method: 'POST',
            body: JSON.stringify({ packages: { npm: created.name, golang: created.full_name } }),
          }).catch(() => {});
          await api(`/api/v1/repos/${created.id}/profile`, { method: 'POST' }).catch(() => {});
          btn.textContent = '3/3 Scanning Issues, Community & Playbooks...';
          await Promise.all([
            api(`/api/v1/repos/${created.id}/discover`, { method: 'POST' }).catch(() => {}),
            api(`/api/v1/repos/${created.id}/community/scan`, { method: 'POST' }).catch(() => {}),
            api(`/api/v1/repos/${created.id}/playbooks`, { method: 'POST' }).catch(() => {}),
          ]);
        }
        closeAddRepoModal();
        await loadAll();
      } catch (err) {
        errEl.textContent = err.message || 'Failed to inspect GitHub repository';
        errEl.classList.remove('hidden');
      } finally {
        btn.disabled = false;
        btn.textContent = 'Audit & Add Live Repo';
      }
    };

    window.triggerAudit = async () => {
      const repo = currentRepo();
      await api(`/api/v1/repos/${repo.id}/audit`, { method: 'POST' });
      await loadAll();
    };

    window.collectMetricsNow = async () => {
      const repo = currentRepo();
      await api(`/api/v1/repos/${repo.id}/metrics/collect`, {
        method: 'POST',
        body: JSON.stringify({ packages: { npm: repo.name, golang: repo.full_name } }),
      });
      await loadAll();
    };

    window.runDiscovery = async () => {
      const repo = currentRepo();
      await api(`/api/v1/repos/${repo.id}/profile`, { method: 'POST' });
      await api(`/api/v1/repos/${repo.id}/discover`, { method: 'POST' });
      await loadAll();
    };

    window.scanCommunityFeeds = async () => {
      const repo = currentRepo();
      await api(`/api/v1/repos/${repo.id}/community/scan`, { method: 'POST' });
      await loadAll();
    };

    window.generatePlaybooks = async () => {
      const repo = currentRepo();
      await api(`/api/v1/repos/${repo.id}/playbooks`, { method: 'POST' });
      await loadAll();
    };

    window.setInboxFilter = (st) => {
      state.inboxStatusFilter = st;
      render();
    };

    window.updateOppStatus = async (id, status) => {
      await api(`/api/v1/opportunities/${id}/status`, {
        method: 'PUT',
        body: JSON.stringify({ status }),
      });
      await loadAll();
    };

    window.saveSettings = async (e) => {
      e.preventDefault();
      await api('/api/v1/settings', {
        method: 'PUT',
        body: JSON.stringify({
          github_token: document.getElementById('setGhToken').value,
          llm_provider: document.getElementById('setLlmProvider').value,
          llm_api_key: document.getElementById('setLlmKey').value,
          slack_webhook_url: document.getElementById('setSlackUrl').value,
        }),
      });
      await loadAll();
    };

    window.saveSetupSettings = async (e) => {
      e.preventDefault();
      await api('/api/v1/settings', {
        method: 'PUT',
        body: JSON.stringify({
          github_token: document.getElementById('setupGhToken').value,
          sonar_token: document.getElementById('setupSonarToken').value,
          sonar_host_url: document.getElementById('setupSonarHost').value,
          sonar_org: document.getElementById('setupSonarOrg').value,
          llm_provider: document.getElementById('setupLlmProvider').value,
          llm_api_key: document.getElementById('setupLlmKey').value,
          slack_webhook_url: document.getElementById('setupSlackUrl').value,
        }),
      });
      const notice = document.getElementById('setupSaveNotice');
      if (notice) {
        notice.classList.remove('hidden');
        setTimeout(() => notice.classList.add('hidden'), 3500);
      }
      await loadAll();
    };

    window.selectPipelinePreset = (type) => {
      state.selectedPipelineType = type;
      render();
    };

    window.installSelectedPipeline = async () => {
      const repo = currentRepo();
      if (!repo) return;
      const pipe = generatePipelineWorkflow(state.selectedPipelineType, repo.full_name);
      const pipeTemplate = PIPELINE_TEMPLATES[state.selectedPipelineType] || {};
      const pipeName = pipeTemplate.name || state.selectedPipelineType;
      document.getElementById('modalTitle').textContent = `Pipeline: ${pipeName}`;
      document.getElementById('modalSubtitle').textContent = `Target: ${pipe.filename} • Repository: ${repo.full_name}`;
      document.getElementById('modalContent').textContent = pipe.content;
      document.getElementById('modalFooter').innerHTML = `
        <button onclick="navigator.clipboard.writeText(document.getElementById('modalContent').textContent)" class="text-xs px-3 py-1.5 rounded-lg bg-slate-800 text-slate-200">Copy YAML</button>
        ${repo.is_owned ? `
          <button id="dispatchPipelineIssueBtn" onclick="dispatchPipelineIssue()" class="text-xs px-3.5 py-1.5 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white font-semibold">1-Click Dispatch Issue (@copilot)</button>
        ` : ''}
      `;
      document.getElementById('modalOverlay').classList.remove('hidden');
    };

    window.dispatchPipelineIssue = async () => {
      const repo = currentRepo();
      if (!repo) return;
      const norm = normalizeRepoConfig(repo);
      const pipe = generatePipelineWorkflow(state.selectedPipelineType, repo.full_name);
      const pipeTemplate = PIPELINE_TEMPLATES[state.selectedPipelineType] || {};
      const pipeName = pipeTemplate.name || state.selectedPipelineType;
      const btn = document.getElementById('dispatchPipelineIssueBtn');
      if (btn) {
        btn.disabled = true;
        btn.textContent = `Filing issue on GitHub for ${norm.copilot_handle}...`;
      }
      const title = `[OSSAlot] Configure ${pipeName} (${pipe.filename})`;
      const body = composeAgentPrompt(repo, {
        title: `Configure ${pipeName} (${pipe.filename})`,
        details: `OSSAlot recommends configuring the **${pipeName}** automation workflow on \`${repo.full_name}\`.\nTarget Workflow File: \`${pipe.filename}\`${pipe.required_secret ? `\nRequired Repository Secret: \`${pipe.required_secret}\`` : ''}`,
        specificationYaml: pipe.content,
      });

      const labels = (repo.issue_labels || 'copilot, pipeline, automation')
        .split(',')
        .map(l => l.trim())
        .filter(Boolean);

      try {
        const res = await api(`/api/v1/repos/${repo.id}/issues`, {
          method: 'POST',
          body: JSON.stringify({
            title,
            body,
            labels,
          }),
        });
        if (res.html_url) {
          document.getElementById('modalSubtitle').innerHTML = `
            <div class="mt-2 p-2.5 rounded-xl bg-indigo-500/20 border border-indigo-500/30 text-indigo-300 font-medium text-xs flex flex-wrap items-center justify-between gap-2">
              <span>&check; GitHub Issue #${res.number} Filed for ${norm.copilot_handle}!</span>
              <a href="${res.html_url}" target="_blank" rel="noopener" class="underline font-bold text-white bg-indigo-600 hover:bg-indigo-500 px-3 py-1 rounded-lg">View Issue on GitHub &nearr;</a>
            </div>
          `;
          if (btn) {
            btn.outerHTML = `<a href="${res.html_url}" target="_blank" rel="noopener" class="text-xs px-3.5 py-1.5 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white font-semibold">Open Issue on GitHub &nearr;</a>`;
          }
        }
      } catch (err) {
        document.getElementById('modalSubtitle').innerHTML = `
          <div class="mt-2 p-2.5 rounded-xl bg-rose-500/20 border border-rose-500/30 text-rose-300 font-medium text-xs">
            &cross; Error filing GitHub Issue: ${err.message || 'Check repository permissions'}
          </div>
        `;
        if (btn) {
          btn.disabled = false;
          btn.textContent = `Retry Dispatch Issue (${norm.copilot_handle})`;
        }
      }
    };

    window.updateRepoPromptPreview = () => {
      const repo = currentRepo();
      if (!repo) return;
      const baseBranch = (document.getElementById('repoConfigBaseBranch') || {}).value || repo.base_branch || repo.default_branch || 'main';
      const branchPrefix = (document.getElementById('repoConfigBranchPrefix') || {}).value || repo.branch_prefix || 'copilot/';
      const copilotHandle = (document.getElementById('repoConfigAgentHandle') || {}).value || repo.copilot_handle || '@copilot';
      const copilotPrompt = (document.getElementById('repoConfigPrompt') || {}).value || repo.copilot_prompt || '';

      const previewRepo = {
        ...repo,
        base_branch: baseBranch,
        branch_prefix: branchPrefix,
        copilot_handle: copilotHandle,
        copilot_prompt: copilotPrompt,
      };

      const previewText = composeAgentPrompt(previewRepo, {
        title: "Remediate Standards & CI Configuration",
        details: "Automated analysis detected missing configurations.",
        specificationYaml: "name: CI Workflow\non: [push, pull_request]",
      });

      const previewEl = document.getElementById('repoConfigPromptPreview');
      if (previewEl) {
        previewEl.textContent = previewText;
      }
    };

    window.appendPromptDirective = (directive) => {
      const el = document.getElementById('repoConfigPrompt');
      if (el) {
        el.value = el.value ? `${el.value.trim()}\n${directive}` : directive;
        window.updateRepoPromptPreview();
      }
    };

    window.onRepoConfigChange = async (repoId) => {
      state.selectedRepoId = repoId;
      await loadAll();
    };

    window.saveRepoConfig = async (e) => {
      e.preventDefault();
      const repo = currentRepo();
      if (!repo) return;

      const isOwned = document.getElementById('repoConfigIsOwned').checked;
      const desc = document.getElementById('repoConfigDesc').value;
      const defBranch = document.getElementById('repoConfigDefBranch').value;
      const baseBranch = document.getElementById('repoConfigBaseBranch').value;
      const branchPrefix = document.getElementById('repoConfigBranchPrefix').value;
      const agentHandle = document.getElementById('repoConfigAgentHandle').value;
      const issueLabels = document.getElementById('repoConfigIssueLabels').value;
      const prompt = document.getElementById('repoConfigPrompt').value;

      const validation = validateRepoConfig({
        base_branch: baseBranch,
        copilot_handle: agentHandle,
        branch_prefix: branchPrefix,
      });

      if (!validation.valid) {
        alert("Configuration Error:\n" + validation.errors.join("\n"));
        return;
      }

      await api(`/api/v1/repos/${repo.id}`, {
        method: 'PATCH',
        body: JSON.stringify({
          is_owned: isOwned,
          description: desc,
          default_branch: defBranch,
          base_branch: baseBranch,
          branch_prefix: branchPrefix,
          copilot_handle: agentHandle,
          issue_labels: issueLabels,
          copilot_prompt: prompt,
        }),
      });

      const notice = document.getElementById('repoConfigSaveNotice');
      if (notice) {
        notice.classList.remove('hidden');
        setTimeout(() => notice.classList.add('hidden'), 3500);
      }
      await loadAll();
    };

    window.dispatchWorkflowPreset = async (presetId) => {
      state.selectedPipelineType = presetId;
      window.switchTab('pipelines');
      window.installSelectedPipeline();
    };

    window.applyAllHygiene = () => {
      const repo = currentRepo();
      const repoLang = repo ? repo.primary_language : 'Go';
      const hygiene = generateHygieneConfigs(repoLang, repo ? repo.full_name : 'project');
      const keys = Object.keys(hygiene);
      const combined = keys.map(k => `### ${hygiene[k].filename} (${hygiene[k].description})\n${hygiene[k].content}`).join('\n\n');
      document.getElementById('modalTitle').textContent = `Code Hygiene Bundle for ${repo ? repo.full_name : 'Project'}`;
      document.getElementById('modalSubtitle').textContent = `Standard configs: .editorconfig, .gitattributes, pre-commit, and ${repoLang} linter.`;
      document.getElementById('modalContent').textContent = combined;
      document.getElementById('modalFooter').innerHTML = `
        <button onclick="navigator.clipboard.writeText(document.getElementById('modalContent').textContent)" class="text-xs px-3 py-1.5 rounded-lg bg-indigo-600 text-white font-medium">Copy All Hygiene Files</button>
      `;
      document.getElementById('modalOverlay').classList.remove('hidden');
    };

    window.checkSonarOnboardingNow = async () => {
      const repo = currentRepo();
      if (!repo) return;
      const [owner, name] = repo.full_name.includes('/') ? repo.full_name.split('/') : ['owner', repo.name];
      const projectKey = `${owner}_${name}`;

      const btn = document.querySelector('button[onclick="checkSonarOnboardingNow()"]');
      if (btn) btn.textContent = 'Checking SonarCloud...';

      let orgRes = null;
      let compRes = null;
      try {
        const [oR, cR] = await Promise.all([
          fetch(`https://sonarcloud.io/api/organizations/search?organizations=${encodeURIComponent(owner)}`).then(r => r.json()).catch(() => null),
          fetch(`https://sonarcloud.io/api/components/show?component=${encodeURIComponent(projectKey)}`).then(r => r.json()).catch(() => null),
        ]);
        orgRes = oR;
        compRes = cR && !cR.errors ? cR : null;

        if (!compRes) {
          const navRes = await fetch(`https://sonarcloud.io/api/navigation/component?component=${encodeURIComponent(projectKey)}`).then(r => r.json()).catch(() => null);
          if (navRes && !navRes.errors) {
            compRes = navRes;
          }
        }
      } finally {
        if (btn) btn.textContent = 'Check Sonar Status for Current Repo';
      }

      state.sonarOnboarding = evaluateSonarOnboarding(repo.full_name, orgRes, compRes);
      render();
    };

    window.runScheduledJob = async (name) => {
      await api(`/api/v1/jobs/${name}/run`, { method: 'POST' });
      await loadAll();
    };

    loadAll();
