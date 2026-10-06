import { spawn } from "node:child_process";
import type * as T from "./types.ts";
import { nonEmpty, parsePrNumber } from "./types.ts";

export interface CommandResult {
  readonly code: number;
  readonly stdout: string;
  readonly stderr: string;
}
export type CommandRunner = (
  argv: readonly [string, ...string[]]
) => Promise<CommandResult>;
export class WatcherQueryError extends Error {
  readonly failure: T.QueryFailure;
  constructor(failure: T.QueryFailure) {
    super(failure.detail);
    this.name = "WatcherQueryError";
    this.failure = failure;
  }
}
export class ChecksUnavailable extends WatcherQueryError {
  constructor(detail: string) {
    super({ kind: "checks-unavailable", retryable: true, detail });
    this.name = "ChecksUnavailable";
  }
}
const firstLine = (value: string): string =>
  value.trim().split(/\r?\n/, 1)[0]?.slice(0, 240) ?? "";
function spawnCommand(
  argv: readonly [string, ...string[]]
): Promise<CommandResult> {
  return new Promise((resolve, reject) => {
    const child = spawn(argv[0], argv.slice(1), {
      stdio: ["ignore", "pipe", "pipe"],
    });
    let stdout = "";
    let stderr = "";
    child.stdout.setEncoding("utf8");
    child.stderr.setEncoding("utf8");
    child.stdout.on("data", (chunk: string) => {
      stdout += chunk;
    });
    child.stderr.on("data", (chunk: string) => {
      stderr += chunk;
    });
    child.on("error", reject);
    child.on("close", (code) => resolve({ code: code ?? -1, stdout, stderr }));
  });
}
function parseJson(text: string, label: string): unknown {
  try {
    return JSON.parse(text);
  } catch (error) {
    throw new WatcherQueryError({
      kind: "json-parse",
      retryable: true,
      detail: `${label}: ${error instanceof Error ? error.message : String(error)}`,
    });
  }
}
function commandExit(code: number, detail: string): never {
  throw new WatcherQueryError({
    kind: "command-exit",
    retryable: true,
    code,
    detail,
  });
}
function raw(value: unknown): string {
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}
function missing(path: string, value?: unknown): never {
  throw new WatcherQueryError({
    kind: "missing-key",
    retryable: true,
    detail:
      value === undefined
        ? `missing ${path}`
        : `invalid ${path}: ${raw(value)}`,
    ...(value === undefined ? {} : { rawValue: raw(value) }),
  });
}
function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
function record(value: unknown, path: string): Record<string, unknown> {
  if (!isRecord(value)) missing(path, value);
  return value;
}
function list(value: unknown, path: string): readonly unknown[] {
  if (!Array.isArray(value)) missing(path, value);
  return value;
}
function at(value: unknown, path: readonly string[]): unknown {
  let current = value;
  for (const key of path) {
    const object = record(current, path.join("."));
    if (!(key in object)) missing(path.join("."));
    current = object[key];
  }
  return current;
}
function string(value: unknown, path: string): string {
  if (typeof value !== "string") missing(path, value);
  return value;
}
const optionalString = (value: unknown, path: string): string | null =>
  value === null ? null : string(value, path);
const displayString = (value: unknown): string =>
  typeof value === "string" ? value : "";
function enumValue<const V extends readonly string[]>(
  value: unknown,
  values: V,
  path: string
): V[number] {
  if (typeof value === "string")
    for (const candidate of values) if (candidate === value) return candidate;
  return missing(path, value);
}
function commentId(value: unknown, path: string): number {
  if (typeof value !== "number" || !Number.isSafeInteger(value) || value <= 0)
    missing(path, value);
  return value;
}
const MERGE_STATE_KEYS = [
  "behind",
  "blocked",
  "clean",
  "dirty",
  "draft",
  "has_hooks",
  "unknown",
  "unstable",
] as const;
const MERGE_STATES: Record<
  (typeof MERGE_STATE_KEYS)[number],
  T.MergeStateStatus
> = {
  behind: "BEHIND",
  blocked: "BLOCKED",
  clean: "CLEAN",
  dirty: "DIRTY",
  draft: "DRAFT",
  has_hooks: "HAS_HOOKS",
  unknown: "UNKNOWN",
  unstable: "UNSTABLE",
};
function parseRemote(value: string): T.Repository | null {
  let normalized = value.trim();
  if (normalized.startsWith("git@github.com:"))
    normalized = `https://github.com/${normalized.slice(15)}`;
  if (normalized.startsWith("ssh://git@github.com/"))
    normalized = `https://github.com/${normalized.slice(21)}`;
  try {
    const url = new URL(normalized);
    const parts = url.pathname
      .replace(/\.git$/, "")
      .split("/")
      .filter(Boolean);
    if (
      url.protocol !== "https:" ||
      url.hostname !== "github.com" ||
      url.port ||
      url.username ||
      url.password ||
      url.search ||
      url.hash ||
      parts.length !== 2
    )
      return null;
    return { owner: parts[0], repo: parts[1] };
  } catch {
    return null;
  }
}
function parsePrUrl(value: string): T.PrContext {
  try {
    const url = new URL(value);
    const parts = url.pathname.split("/").filter(Boolean);
    if (
      url.protocol !== "https:" ||
      url.hostname !== "github.com" ||
      url.port ||
      url.username ||
      url.password ||
      url.search ||
      url.hash ||
      parts.length !== 4 ||
      parts[2] !== "pull"
    )
      throw new Error("not a canonical GitHub pull URL");
    return {
      owner: parts[0],
      repo: parts[1],
      number: parsePrNumber(Number(parts[3])),
    };
  } catch (error) {
    throw new WatcherQueryError({
      kind: "invalid-context-url",
      retryable: false,
      rawValue: value,
      detail: `could not infer owner/repo from PR URL: ${value} (${error instanceof Error ? error.message : String(error)})`,
    });
  }
}
interface CheckDetails {
  readonly name: string;
  readonly description: string;
  readonly link: string;
  readonly workflow: string;
}
// The owner-approval gate is excluded from pending everywhere, so the rule has
// one home. Classifying it as pending on either read path makes the watcher
// wait on a human, which is the behaviour #172004 removed from the Python.
function pendingOrGate(details: CheckDetails, reportedState: string): T.Check {
  return details.name === "Code Review Gate"
    ? {
        ...details,
        kind: "code-review-gate",
        name: "Code Review Gate",
        reportedState,
      }
    : { ...details, kind: "pending", reportedState };
}
export function parseCheckRun(value: unknown): T.Check {
  const run = record(value, "check run");
  const details = {
    name: string(run.name, "check run.name"),
    description: isRecord(run.output) ? displayString(run.output.title) : "",
    link: displayString(run.details_url) || displayString(run.html_url),
    workflow: "",
  };
  const status = string(run.status, "check run.status").toUpperCase();
  const conclusion = displayString(run.conclusion).toUpperCase();
  if (status !== "COMPLETED") return pendingOrGate(details, "PENDING");
  if (conclusion === "SUCCESS")
    return { ...details, kind: "passed", reportedState: "SUCCESS" };
  if (conclusion === "NEUTRAL" || conclusion === "SKIPPED")
    return { ...details, kind: "skipped", reportedState: conclusion };
  return {
    ...details,
    kind: "failed",
    reportedState: conclusion === "ACTION_REQUIRED" ? conclusion : "FAILURE",
  };
}
export function parseCommitStatus(value: unknown): T.Check {
  const status = record(value, "commit status");
  const details = {
    name: string(status.context, "commit status.context"),
    description: displayString(status.description),
    link: displayString(status.target_url),
    workflow: "",
  };
  const state = string(status.state, "commit status.state").toUpperCase();
  if (state === "PENDING" || state === "EXPECTED")
    return pendingOrGate(details, "PENDING");
  return state === "SUCCESS"
    ? { ...details, kind: "passed", reportedState: state }
    : { ...details, kind: "failed", reportedState: state || "FAILURE" };
}
export function deriveRollupState(checks: readonly T.Check[]): T.RollupState {
  const failed = checks.filter((check) => check.kind === "failed");
  if (failed.some((check) => check.reportedState !== "ERROR")) return "FAILURE";
  if (failed.length > 0) return "ERROR";
  if (
    checks.some(
      (check) => check.kind === "pending" || check.kind === "code-review-gate"
    )
  )
    return "PENDING";
  return checks.length > 0 ? "SUCCESS" : null;
}
function parseComment(value: unknown): T.ReviewComment {
  const object = record(value, "review comment");
  const user =
    object.user === null ? null : record(object.user, "review comment.user");
  const line = object.line ?? object.original_line ?? null;
  return {
    authorLogin:
      user === null
        ? null
        : optionalString(user.login, "review comment.user.login"),
    body: string(object.body, "review comment.body"),
    path: optionalString(object.path, "review comment.path"),
    line:
      line === null
        ? null
        : Number.isInteger(line)
          ? Number(line)
          : missing("review comment.line", line),
    createdAt: string(object.created_at, "review comment.created_at"),
  };
}
function isBugbot(comment: T.ReviewComment | null): boolean {
  if (comment === null) return false;
  const author = (comment.authorLogin ?? "").toLowerCase();
  const body = comment.body.toLowerCase();
  return (
    author.includes("bugbot") ||
    (author === "cursor" &&
      [
        "bugbot",
        "cursor_automation_id",
        "agentic security review",
        "description start",
        "severity",
      ].some((token) => body.includes(token)))
  );
}
function passKey(comment: T.ReviewComment | null): string | null {
  if (comment === null) return null;
  for (const pattern of [
    /RUN_ID:\s*([a-zA-Z0-9_.:-]+)/,
    /CURSOR_AUTOMATION_ID:\s*([a-zA-Z0-9_.:-]+)/,
  ]) {
    const match = pattern.exec(comment.body);
    if (match?.[1]) return match[1];
  }
  return null;
}
export function parseCcrThreads(
  rows: unknown,
  comments: readonly unknown[]
): readonly T.ReviewThread[] {
  const byId = new Map<number, T.ReviewComment>();
  for (const item of comments)
    byId.set(
      commentId(record(item, "review comment").id, "review comment.id"),
      parseComment(item)
    );
  const threads = list(rows, "review threads").map((row) => {
    const thread = record(row, "review thread");
    if (typeof thread.resolved !== "boolean")
      missing("review thread.resolved", thread.resolved);
    const ids = list(thread.comment_ids, "review thread.comment_ids");
    if (ids.length === 0) missing("review thread.comment_ids", ids);
    const opening = commentId(ids[0], "review thread.comment_ids[0]");
    return {
      id: String(opening),
      firstComment: byId.get(opening) ?? null,
      resolved: thread.resolved,
    };
  });
  const keys = new Set<string>();
  let keyless = false;
  for (const thread of threads) {
    if (!isBugbot(thread.firstComment)) continue;
    const key = passKey(thread.firstComment);
    if (key === null) keyless = true;
    else keys.add(key);
  }
  const passes = keys.size > 0 ? keys.size : keyless ? 1 : 0;
  return threads
    .filter((thread) => !thread.resolved)
    .map(({ id, firstComment }) => ({
      id,
      firstComment,
      isBugbot: isBugbot(firstComment),
      bugbotReviewPasses: passes,
    }));
}
const DECIDING_REVIEW_STATES = new Set([
  "APPROVED",
  "CHANGES_REQUESTED",
  "DISMISSED",
]);
// REST has no reviewDecision, and REVIEW_REQUIRED would need branch-protection
// reads. Policy gates only on CHANGES_REQUESTED, and mergeStateStatus BLOCKED
// already covers a missing required review.
export function deriveReviewDecision(
  reviews: readonly unknown[]
): T.ReviewDecision {
  const latest = new Map<string, string>();
  for (const item of reviews) {
    const review = record(item, "review");
    const state = string(review.state, "review.state");
    if (!DECIDING_REVIEW_STATES.has(state)) continue;
    const user =
      review.user === null ? null : record(review.user, "review.user");
    latest.set(
      user === null ? "" : string(user.login, "review.user.login"),
      state
    );
  }
  const states = [...latest.values()];
  if (states.includes("CHANGES_REQUESTED")) return "CHANGES_REQUESTED";
  return states.includes("APPROVED") ? "APPROVED" : null;
}
export function parsePullRequestRest(
  value: unknown,
  reviewDecision: T.ReviewDecision,
  context: T.PrContext
): T.PullRequestFacts {
  const pull = record(value, "pull request");
  const head = record(pull.head, "pull request.head");
  const base = record(pull.base, "pull request.base");
  if (typeof pull.draft !== "boolean")
    missing("pull request.draft", pull.draft);
  const mergedAt = optionalString(pull.merged_at, "pull request.merged_at");
  const state = enumValue(
    pull.state,
    ["open", "closed"] as const,
    "pull request.state"
  );
  return {
    context,
    mergeable:
      pull.mergeable === null
        ? "UNKNOWN"
        : pull.mergeable === true
          ? "MERGEABLE"
          : pull.mergeable === false
            ? "CONFLICTING"
            : missing("pull request.mergeable", pull.mergeable),
    mergeStateStatus:
      MERGE_STATES[
        enumValue(
          pull.mergeable_state,
          MERGE_STATE_KEYS,
          "pull request.mergeable_state"
        )
      ],
    reviewDecision,
    headRefOid: optionalString(head.sha, "pull request.head.sha"),
    headRefName: string(head.ref, "pull request.head.ref"),
    baseRefName: string(base.ref, "pull request.base.ref"),
    state:
      state === "open"
        ? "OPEN"
        : mergedAt !== null || pull.merged === true
          ? "MERGED"
          : "CLOSED",
    mergedAt,
    isDraft: pull.draft,
  };
}
const PER_PAGE = 100;
const MAX_PAGES = 50;
// Bounds the REST calls per poll. Policy reads only the head's rollup and
// whether any earlier commit passed, so older commits add nothing.
const MAX_PREVIOUS_COMMITS = 10;
type PageItems = (page: unknown, path: string) => readonly unknown[];
const arrayItems: PageItems = (page, path) => list(page, path);
const itemsAt =
  (key: string): PageItems =>
  (page, path) =>
    list(at(page, [key]), `${path}.${key}`);
const checkRunItems = itemsAt("check_runs");
// `.state` of the combined status reads "pending" for a commit that has no
// statuses at all, so only the listed statuses count.
const statusItems = itemsAt("statuses");
function tooManyPages(path: string): never {
  throw new WatcherQueryError({
    kind: "missing-key",
    retryable: true,
    detail: `${path}: more than ${MAX_PAGES} full pages`,
  });
}
const repoPath = (repository: T.Repository): string =>
  `repos/${repository.owner}/${repository.repo}`;
const pullPath = (context: T.PrContext): string =>
  `${repoPath(context)}/pulls/${context.number}`;
function cursorPage(after: string | null): number {
  if (after === null) return 1;
  if (!/^[1-9][0-9]*$/.test(after)) missing("check rollup cursor", after);
  return Number(after);
}

export class GhGitHubReader implements T.GitHubReader {
  readonly #run: CommandRunner;
  constructor(run: CommandRunner = spawnCommand) {
    this.#run = run;
  }
  async #json(path: string): Promise<unknown> {
    const argv = ["gh", "api", path] as const;
    const result = await this.#run(argv);
    if (result.code !== 0)
      commandExit(
        result.code,
        firstLine(result.stderr) || `${argv.join(" ")} exited ${result.code}`
      );
    return parseJson(result.stdout, argv.join(" "));
  }
  async #page(
    path: string,
    page: number,
    items: PageItems
  ): Promise<readonly unknown[]> {
    if (page > MAX_PAGES) tooManyPages(path);
    const separator = path.includes("?") ? "&" : "?";
    const url = `${path}${separator}per_page=${PER_PAGE}&page=${page}`;
    return items(await this.#json(url), url);
  }
  // Pages are walked by number because `gh api --paginate` follows Link
  // headers to /repositories/{id}/..., which the cloud egress proxy refuses.
  async #all(
    path: string,
    items: PageItems = arrayItems
  ): Promise<readonly unknown[]> {
    const collected: unknown[] = [];
    for (let page = 1; ; page++) {
      const batch = await this.#page(path, page, items);
      collected.push(...batch);
      if (batch.length < PER_PAGE) return collected;
    }
  }
  async #headSha(context: T.PrContext): Promise<string> {
    return string(
      at(await this.#json(pullPath(context)), ["head", "sha"]),
      "pull request.head.sha"
    );
  }
  async #commitChecks(
    repository: T.Repository,
    sha: string
  ): Promise<readonly T.Check[]> {
    const commit = `${repoPath(repository)}/commits/${sha}`;
    const runs = await this.#all(`${commit}/check-runs`, checkRunItems);
    const statuses = await this.#all(`${commit}/status`, statusItems);
    return [...runs.map(parseCheckRun), ...statuses.map(parseCommitStatus)];
  }
  async originRepo(): Promise<T.Repository | null> {
    const result = await this.#run(["git", "remote", "get-url", "origin"]);
    return result.code === 0 ? parseRemote(result.stdout) : null;
  }
  async currentPr(pr: T.PrNumber | null): Promise<T.PrContext> {
    const origin = await this.originRepo();
    if (origin === null)
      return commandExit(1, "git remote origin is not a GitHub repository");
    if (pr !== null) {
      const pull = record(
        await this.#json(`${repoPath(origin)}/pulls/${pr}`),
        "current PR"
      );
      return {
        ...parsePrUrl(string(pull.html_url, "current PR.html_url")),
        number: pr,
      };
    }
    const branch = await this.#run(["git", "branch", "--show-current"]);
    const name = branch.code === 0 ? branch.stdout.trim() : "";
    if (name === "")
      return commandExit(
        branch.code || 1,
        firstLine(branch.stderr) || "could not determine the current branch"
      );
    const head = `${origin.owner}:${encodeURIComponent(name)}`;
    const [first] = list(
      await this.#json(`${repoPath(origin)}/pulls?state=open&head=${head}`),
      "current PR"
    );
    if (first === undefined)
      return commandExit(
        1,
        `no open pull requests found for branch "${name}"`
      );
    const pull = record(first, "current PR");
    return {
      ...parsePrUrl(string(pull.html_url, "current PR.html_url")),
      number: parsePrNumber(pull.number, "current PR.number"),
    };
  }
  async pullRequest(context: T.PrContext): Promise<T.PullRequestFacts> {
    const pull = await this.#json(pullPath(context));
    const reviews = await this.#all(`${pullPath(context)}/reviews`);
    return parsePullRequestRest(pull, deriveReviewDecision(reviews), context);
  }
  async openPullRequests(
    repository: T.Repository
  ): Promise<readonly T.OpenPullRequest[]> {
    const pulls = await this.#all(`${repoPath(repository)}/pulls?state=open`);
    return pulls.map((item, index) => {
      const object = record(item, `open PRs[${index}]`);
      return {
        number: parsePrNumber(object.number, `open PRs[${index}].number`),
        headRefName: string(
          at(object, ["head", "ref"]),
          `open PRs[${index}].head.ref`
        ),
        baseRefName: string(
          at(object, ["base", "ref"]),
          `open PRs[${index}].base.ref`
        ),
      };
    });
  }
  async checksFastPath(context: T.PrContext): Promise<T.ChecksFastPath> {
    try {
      const sha = await this.#headSha(context);
      const checks = await this.#commitChecks(context, sha);
      return { kind: "checks", checks };
    } catch (error) {
      if (
        !(error instanceof WatcherQueryError) ||
        error.failure.kind !== "command-exit"
      )
        throw error;
      return {
        kind: "unusable",
        exitCode: error.failure.code,
        stderr: error.failure.detail,
      };
    }
  }
  async checkRollupPage(
    context: T.PrContext,
    after: string | null
  ): Promise<T.RollupPage> {
    const page = cursorPage(after);
    const sha = await this.#headSha(context);
    const commit = `${repoPath(context)}/commits/${sha}`;
    const runs = await this.#page(`${commit}/check-runs`, page, checkRunItems);
    const statuses =
      page === 1 ? await this.#all(`${commit}/status`, statusItems) : [];
    return {
      checks: [...runs.map(parseCheckRun), ...statuses.map(parseCommitStatus)],
      endCursor: runs.length === PER_PAGE ? String(page + 1) : null,
    };
  }
  async reviewThreads(
    context: T.PrContext
  ): Promise<readonly T.ReviewThread[]> {
    // Only the cloud egress proxy serves ccr/ routes. github.com answers 404.
    const rows = await this.#json(`${pullPath(context)}/ccr/review_threads`);
    const comments = await this.#all(`${pullPath(context)}/comments`);
    return parseCcrThreads(rows, comments);
  }
  async commitRollups(
    context: T.PrContext
  ): Promise<readonly T.CommitRollup[]> {
    const commits = await this.#all(`${pullPath(context)}/commits`);
    const rollups: T.CommitRollup[] = [];
    for (let index = commits.length - 1; index >= 0; index--) {
      const oid = string(
        record(commits[index], `commits[${index}]`).sha,
        `commits[${index}].sha`
      );
      const state = deriveRollupState(await this.#commitChecks(context, oid));
      rollups.push({ oid, state });
      if (rollups.length > MAX_PREVIOUS_COMMITS) break;
      if (rollups.length > 1 && state === "SUCCESS") break;
    }
    return rollups;
  }
}

export async function resolveChecks(
  reader: T.GitHubReader,
  context: T.PrContext
): Promise<T.CheckRead> {
  const fast = await reader.checksFastPath(context);
  const direct = fast.kind === "checks" ? nonEmpty(fast.checks) : null;
  if (direct !== null) return { source: "gh-pr-checks", checks: direct };
  const checks: T.Check[] = [];
  let after: string | null = null;
  do {
    const page = await reader.checkRollupPage(context, after);
    checks.push(...page.checks);
    after = page.endCursor;
  } while (after !== null);
  const fallback = nonEmpty(checks);
  if (fallback !== null) return { source: "graphql-rollup", checks: fallback };
  const suffix =
    fast.kind === "unusable"
      ? `fast path exit=${fast.exitCode}; paged rollup was empty${firstLine(fast.stderr) ? `; ${firstLine(fast.stderr)}` : ""}`
      : "fast path and paged rollup were empty";
  throw new ChecksUnavailable(`could not read PR checks: ${suffix}`);
}
export async function resolveContext(args: {
  readonly reader: T.GitHubReader;
  readonly owner: string | null;
  readonly repo: string | null;
  readonly pr: T.PrNumber | null;
}): Promise<T.PrContext> {
  if (args.pr !== null && args.owner !== null && args.repo !== null)
    return { owner: args.owner, repo: args.repo, number: args.pr };
  if (args.pr !== null) {
    const origin = await args.reader.originRepo();
    if (origin !== null)
      return {
        owner: args.owner ?? origin.owner,
        repo: args.repo ?? origin.repo,
        number: args.pr,
      };
  }
  const inferred = await args.reader.currentPr(args.pr);
  return {
    owner: args.owner ?? inferred.owner,
    repo: args.repo ?? inferred.repo,
    number: args.pr ?? inferred.number,
  };
}
export function orderStack(
  context: T.PrContext,
  open: readonly T.OpenPullRequest[]
): T.NonEmpty<T.PrContext> {
  const byNumber = new Map(open.map((pr) => [pr.number, pr]));
  const byHead = new Map(open.map((pr) => [pr.headRefName, pr]));
  const children = new Map<string, T.OpenPullRequest[]>();
  for (const pr of open)
    children.set(pr.baseRefName, [...(children.get(pr.baseRefName) ?? []), pr]);
  for (const values of children.values())
    values.sort((a, b) => a.number - b.number);
  const start = byNumber.get(context.number);
  if (start === undefined) return [context];
  const down: T.OpenPullRequest[] = [];
  let current = start;
  while (byHead.has(current.baseRefName)) {
    const parent = byHead.get(current.baseRefName);
    if (parent === undefined) break;
    down.push(parent);
    current = parent;
  }
  const seen = new Set<T.PrNumber>([
    ...down.map((pr) => pr.number),
    start.number,
  ]);
  const up: T.OpenPullRequest[] = [];
  const visit = (parent: T.OpenPullRequest): void => {
    for (const child of children.get(parent.headRefName) ?? []) {
      if (seen.has(child.number)) continue;
      seen.add(child.number);
      up.push(child);
      visit(child);
    }
  };
  visit(start);
  return (
    nonEmpty(
      [...down.reverse(), start, ...up].map((pr) => ({
        ...context,
        number: pr.number,
      }))
    ) ?? [context]
  );
}
export async function discoverStack(
  reader: T.GitHubReader,
  context: T.PrContext
): Promise<T.NonEmpty<T.PrContext>> {
  return orderStack(context, await reader.openPullRequests(context));
}
