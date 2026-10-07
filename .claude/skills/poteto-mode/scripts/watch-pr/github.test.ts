import { describe, expect, it } from "bun:test";
import {
  ChecksUnavailable,
  GhGitHubReader,
  WatcherQueryError,
  deriveReviewDecision,
  deriveRollupState,
  orderStack,
  parseCcrThreads,
  parseCheckRun,
  parseCommitStatus,
  parsePullRequestRest,
  resolveChecks,
  resolveContext,
  type CommandRunner,
} from "./github.ts";
import {
  fakeReader,
  failedCheck,
  passingCheck,
  pendingCheck,
} from "./fakes.test-helper.ts";
import { readSnapshot } from "./policy.ts";
import type { QueryFailure } from "./types.ts";
import { parsePrNumber } from "./types.ts";

const context = {
  owner: "owner",
  repo: "repo",
  number: parsePrNumber(42),
};

describe("checks fallback chain", () => {
  it("uses a non-empty fast-path result without a rollup query", async () => {
    const reader = fakeReader({
      fastPath: { kind: "checks", checks: [passingCheck("fast")] },
    });
    const read = await resolveChecks(reader, context);
    expect(read.source).toBe("gh-pr-checks");
    expect(read.checks.map((check) => check.name)).toEqual(["fast"]);
    expect(reader.calls).toEqual(["checksFastPath"]);
  });

  it("pages the rollup fallback when the fast path is unusable", async () => {
    const reader = fakeReader({
      fastPath: { kind: "unusable", exitCode: 8, stderr: "" },
      rollupPages: [
        { checks: [passingCheck("first")], endCursor: "next" },
        { checks: [failedCheck("second")], endCursor: null },
      ],
    });
    const read = await resolveChecks(reader, context);
    expect(read.source).toBe("graphql-rollup");
    expect(read.checks.map((check) => check.name)).toEqual(["first", "second"]);
    expect(reader.calls).toEqual([
      "checksFastPath",
      "checkRollupPage:null",
      "checkRollupPage:next",
    ]);
  });

  it("falls back when valid fast-path JSON represented an empty list", async () => {
    const reader = fakeReader({
      fastPath: { kind: "checks", checks: [] },
      rollupPages: [{ checks: [pendingCheck("fallback")], endCursor: null }],
    });
    expect((await resolveChecks(reader, context)).checks[0].name).toBe(
      "fallback"
    );
    expect(reader.calls).toEqual(["checksFastPath", "checkRollupPage:null"]);
  });

  it("fails closed when both paths are empty", async () => {
    const reader = fakeReader({
      fastPath: {
        kind: "unusable",
        exitCode: 8,
        stderr: "credential cannot read checks",
      },
    });
    await expect(resolveChecks(reader, context)).rejects.toBeInstanceOf(
      ChecksUnavailable
    );
    expect(reader.calls).toEqual(["checksFastPath", "checkRollupPage:null"]);
  });
});

class Exit {
  constructor(
    readonly code: number,
    readonly stderr: string
  ) {}
}
type Route = (call: string) => unknown;
function fakeGh(routes: Record<string, unknown> | Route) {
  const calls: string[] = [];
  const lookup: Route =
    typeof routes === "function" ? routes : (call) => routes[call];
  const run: CommandRunner = async (argv) => {
    const call = argv.join(" ");
    if (/graphql|--paginate|^gh pr /i.test(call))
      throw new Error(`forbidden command: ${call}`);
    calls.push(call);
    const response = lookup(call);
    if (response === undefined)
      return { code: 1, stdout: "", stderr: "gh: Not Found (HTTP 404)\n" };
    if (response instanceof Exit)
      return { code: response.code, stdout: "", stderr: response.stderr };
    return {
      code: 0,
      stdout:
        typeof response === "string" ? response : JSON.stringify(response),
      stderr: "",
    };
  };
  return { reader: new GhGitHubReader(run), calls };
}
async function failureOf(action: () => unknown): Promise<QueryFailure> {
  try {
    await action();
  } catch (error) {
    if (error instanceof WatcherQueryError) return error.failure;
    throw error;
  }
  throw new Error("expected a WatcherQueryError");
}

const api = "gh api repos/owner/repo";
function restPull(overrides: Record<string, unknown> = {}) {
  return {
    url: "https://api.github.com/repos/owner/repo/pulls/42",
    html_url: "https://github.com/owner/repo/pull/42",
    number: 42,
    state: "open",
    locked: false,
    title: "Add a thing",
    user: { login: "octo", id: 1, type: "User" },
    draft: false,
    merged: false,
    merged_at: null,
    mergeable: true,
    rebaseable: true,
    mergeable_state: "clean",
    head: {
      label: "owner:feature",
      ref: "feature",
      sha: "c3",
      repo: { full_name: "owner/repo" },
    },
    base: {
      label: "owner:main",
      ref: "main",
      sha: "b0",
      repo: { full_name: "owner/repo" },
    },
    ...overrides,
  };
}
function restReview(login: string | null, state: string, id: number) {
  return {
    id,
    user: login === null ? null : { login, id, type: "User" },
    body: "",
    state,
    submitted_at: `2026-10-0${id}T10:00:00Z`,
    commit_id: "c3",
  };
}
function restCheckRun(
  name: string,
  status: string,
  conclusion: string | null,
  overrides: Record<string, unknown> = {}
) {
  return {
    id: 7,
    head_sha: "c3",
    name,
    status,
    conclusion,
    html_url: `https://github.com/owner/repo/runs/${name}`,
    details_url: `https://github.com/owner/repo/actions/runs/1/job/${name}`,
    started_at: "2026-10-06T10:00:00Z",
    completed_at: status === "completed" ? "2026-10-06T10:05:00Z" : null,
    output: { title: null, summary: null, annotations_count: 0 },
    app: { slug: "github-actions" },
    ...overrides,
  };
}
const checkRuns = (runs: readonly unknown[]) => ({
  total_count: runs.length,
  check_runs: runs,
});
function restStatus(context: string, state: string) {
  return {
    id: 9,
    context,
    state,
    description: `${context} is ${state}`,
    target_url: `https://ci.example/${context}`,
    created_at: "2026-10-06T10:00:00Z",
  };
}
const combinedStatus = (state: string, statuses: readonly unknown[]) => ({
  state,
  sha: "c3",
  total_count: statuses.length,
  statuses,
});
function restComment(
  id: number,
  login: string | null,
  body: string,
  overrides: Record<string, unknown> = {}
) {
  return {
    id,
    pull_request_review_id: 1,
    user: login === null ? null : { login, id, type: "User" },
    body,
    path: "a.ts",
    line: 3,
    original_line: 3,
    created_at: "2026-10-06T10:00:00Z",
    ...overrides,
  };
}

describe("pull request facts from REST", () => {
  it("maps a REST pull request to facts", () => {
    expect(
      parsePullRequestRest(
        restPull({ mergeable: null, mergeable_state: "blocked", draft: true }),
        "APPROVED",
        context
      )
    ).toEqual({
      context,
      mergeable: "UNKNOWN",
      mergeStateStatus: "BLOCKED",
      reviewDecision: "APPROVED",
      headRefOid: "c3",
      headRefName: "feature",
      baseRefName: "main",
      state: "OPEN",
      mergedAt: null,
      isDraft: true,
    });
  });

  it("maps mergeable true, false and null", () => {
    const cases = [
      [true, "MERGEABLE"],
      [false, "CONFLICTING"],
      [null, "UNKNOWN"],
    ] as const;
    for (const [mergeable, expected] of cases)
      expect(
        parsePullRequestRest(restPull({ mergeable }), null, context).mergeable
      ).toBe(expected);
  });

  it("uppercases every REST mergeable_state", () => {
    const states = [
      "behind",
      "blocked",
      "clean",
      "dirty",
      "draft",
      "has_hooks",
      "unknown",
      "unstable",
    ];
    expect(
      states.map(
        (mergeable_state) =>
          parsePullRequestRest(restPull({ mergeable_state }), null, context)
            .mergeStateStatus
      )
    ).toEqual([
      "BEHIND",
      "BLOCKED",
      "CLEAN",
      "DIRTY",
      "DRAFT",
      "HAS_HOOKS",
      "UNKNOWN",
      "UNSTABLE",
    ]);
  });

  it("tells a merged PR from one closed without merging", () => {
    const merged = parsePullRequestRest(
      restPull({
        state: "closed",
        merged: true,
        merged_at: "2026-10-06T12:00:00Z",
      }),
      null,
      context
    );
    expect([merged.state, merged.mergedAt]).toEqual([
      "MERGED",
      "2026-10-06T12:00:00Z",
    ]);
    const { merged: _, ...withoutMergedFlag } = restPull({
      state: "closed",
      merged_at: "2026-10-06T12:00:00Z",
    });
    expect(
      parsePullRequestRest(withoutMergedFlag, null, context).state
    ).toBe("MERGED");
    const closed = parsePullRequestRest(
      restPull({ state: "closed", merged: false, merged_at: null }),
      null,
      context
    );
    expect([closed.state, closed.mergedAt]).toEqual(["CLOSED", null]);
  });

  it("fails closed on unknown enum values with the raw value", async () => {
    expect(
      await failureOf(() =>
        parsePullRequestRest(
          restPull({ mergeable_state: "future_state" }),
          null,
          context
        )
      )
    ).toEqual({
      kind: "missing-key",
      retryable: true,
      detail: 'invalid pull request.mergeable_state: "future_state"',
      rawValue: '"future_state"',
    });
    expect(
      await failureOf(() =>
        parsePullRequestRest(restPull({ mergeable: "yes" }), null, context)
      )
    ).toMatchObject({ kind: "missing-key", rawValue: '"yes"' });
  });
});

describe("review decision from REST reviews", () => {
  it("derives the decision from each reviewer's latest deciding review", () => {
    const cases = [
      [
        [restReview("alice", "APPROVED", 1), restReview("alice", "CHANGES_REQUESTED", 2)],
        "CHANGES_REQUESTED",
      ],
      [
        [restReview("alice", "CHANGES_REQUESTED", 1), restReview("alice", "APPROVED", 2)],
        "APPROVED",
      ],
      [
        [restReview("alice", "DISMISSED", 1), restReview("bob", "COMMENTED", 2)],
        null,
      ],
      [
        [restReview("alice", "CHANGES_REQUESTED", 1), restReview("alice", "DISMISSED", 2)],
        null,
      ],
      [
        [restReview("alice", "APPROVED", 1), restReview("bob", "CHANGES_REQUESTED", 2)],
        "CHANGES_REQUESTED",
      ],
      [
        [restReview("alice", "APPROVED", 1), restReview("alice", "COMMENTED", 2)],
        "APPROVED",
      ],
      [[restReview("alice", "COMMENTED", 1)], null],
      [[], null],
    ] as const;
    for (const [reviews, expected] of cases)
      expect(deriveReviewDecision(reviews)).toBe(expected);
  });
});

describe("checks from REST check runs and statuses", () => {
  it("classifies check runs by status and conclusion", () => {
    const cases = [
      ["queued", null, "pending", "PENDING"],
      ["in_progress", null, "pending", "PENDING"],
      ["completed", "success", "passed", "SUCCESS"],
      ["completed", "neutral", "skipped", "NEUTRAL"],
      ["completed", "skipped", "skipped", "SKIPPED"],
      ["completed", "action_required", "failed", "ACTION_REQUIRED"],
      ["completed", "failure", "failed", "FAILURE"],
      ["completed", "timed_out", "failed", "FAILURE"],
      ["completed", "cancelled", "failed", "FAILURE"],
      ["completed", "future_value", "failed", "FAILURE"],
    ] as const;
    expect(
      cases.map(([status, conclusion]) => {
        const check = parseCheckRun(restCheckRun("ci", status, conclusion));
        return [check.kind, check.reportedState];
      })
    ).toEqual(cases.map(([, , kind, reportedState]) => [kind, reportedState]));
  });

  it("takes the link from details_url, then html_url, and the title as description", () => {
    expect(
      parseCheckRun(
        restCheckRun("build", "completed", "failure", {
          details_url: null,
          output: { title: "2 tests failed", summary: "" },
        })
      )
    ).toEqual({
      kind: "failed",
      name: "build",
      reportedState: "FAILURE",
      description: "2 tests failed",
      link: "https://github.com/owner/repo/runs/build",
      workflow: "",
    });
  });

  it("classifies commit statuses", () => {
    expect(
      ["pending", "success", "failure", "error"].map((state) => {
        const check = parseCommitStatus(restStatus("ci/legacy", state));
        return [check.kind, check.reportedState];
      })
    ).toEqual([
      ["pending", "PENDING"],
      ["passed", "SUCCESS"],
      ["failed", "FAILURE"],
      ["failed", "ERROR"],
    ]);
  });

  it("classifies a pending Code Review Gate as the gate on both shapes", () => {
    expect(
      parseCheckRun(restCheckRun("Code Review Gate", "in_progress", null)).kind
    ).toBe("code-review-gate");
    expect(
      parseCommitStatus(restStatus("Code Review Gate", "pending")).kind
    ).toBe("code-review-gate");
  });

  it("rejects a check run without a status", async () => {
    expect(
      await failureOf(() => parseCheckRun({ name: "ci", conclusion: null }))
    ).toEqual({
      kind: "missing-key",
      retryable: true,
      detail: "missing check run.status",
    });
  });
});

describe("commit rollup state", () => {
  it("derives the rollup like GitHub's statusCheckRollup", () => {
    const cases = [
      [[], null],
      [[passingCheck("a"), parseCheckRun(restCheckRun("b", "completed", "skipped"))], "SUCCESS"],
      [[passingCheck("a"), pendingCheck("b")], "PENDING"],
      [[pendingCheck("a"), parseCommitStatus(restStatus("s", "error"))], "ERROR"],
      [
        [parseCommitStatus(restStatus("s", "error")), failedCheck("b")],
        "FAILURE",
      ],
      [[parseCheckRun(restCheckRun("a", "completed", "action_required"))], "FAILURE"],
    ] as const;
    for (const [checks, expected] of cases)
      expect(deriveRollupState(checks)).toBe(expected);
  });
});

describe("review threads from ccr rows and REST comments", () => {
  const rows = [
    { comment_ids: [101, 102], resolved: false, outdated: false, extra: 1 },
    { comment_ids: [201], resolved: false, outdated: true },
    { comment_ids: [301], resolved: true, outdated: false },
    { comment_ids: [401], resolved: false, outdated: false },
    { comment_ids: [501], resolved: false, outdated: false },
  ];
  const comments = [
    restComment(101, "bugbot", "RUN_ID: run-1"),
    restComment(102, "octo", "fixed"),
    restComment(201, "cursor", "CURSOR_AUTOMATION_ID: run-2 severity high", {
      path: null,
      line: null,
      original_line: null,
    }),
    restComment(301, "bugbot", "RUN_ID: run-3"),
    restComment(401, null, "nit", { line: null, original_line: 7 }),
  ];

  it("keeps unresolved threads keyed by opening comment and counts Bugbot passes over all", () => {
    expect(parseCcrThreads(rows, comments)).toEqual([
      {
        id: "101",
        firstComment: {
          authorLogin: "bugbot",
          body: "RUN_ID: run-1",
          path: "a.ts",
          line: 3,
          createdAt: "2026-10-06T10:00:00Z",
        },
        isBugbot: true,
        bugbotReviewPasses: 3,
      },
      {
        id: "201",
        firstComment: {
          authorLogin: "cursor",
          body: "CURSOR_AUTOMATION_ID: run-2 severity high",
          path: null,
          line: null,
          createdAt: "2026-10-06T10:00:00Z",
        },
        isBugbot: true,
        bugbotReviewPasses: 3,
      },
      {
        id: "401",
        firstComment: {
          authorLogin: null,
          body: "nit",
          path: "a.ts",
          line: 7,
          createdAt: "2026-10-06T10:00:00Z",
        },
        isBugbot: false,
        bugbotReviewPasses: 3,
      },
      {
        id: "501",
        firstComment: null,
        isBugbot: false,
        bugbotReviewPasses: 3,
      },
    ]);
  });

  it("fails closed on shapes it does not know instead of reading no threads", async () => {
    const cases = [
      [{ message: "Not Found" }, 'invalid review threads: {"message":"Not Found"}'],
      [[{ resolved: false }], "missing review thread.comment_ids"],
      [[{ comment_ids: [1], resolved: "no" }], 'invalid review thread.resolved: "no"'],
      [[{ comment_ids: [], resolved: false }], "invalid review thread.comment_ids: []"],
      [[{ comment_ids: ["1"], resolved: false }], 'invalid review thread.comment_ids[0]: "1"'],
    ] as const;
    for (const [value, detail] of cases)
      expect(await failureOf(() => parseCcrThreads(value, []))).toMatchObject({
        kind: "missing-key",
        detail,
      });
  });
});

describe("GhGitHubReader over gh api REST", () => {
  it("reads pull request facts from pulls/{n} and its reviews", async () => {
    const gh = fakeGh({
      [`${api}/pulls/42`]: restPull({
        mergeable: false,
        mergeable_state: "dirty",
      }),
      [`${api}/pulls/42/reviews?per_page=100&page=1`]: [
        restReview("alice", "APPROVED", 1),
        restReview("alice", "CHANGES_REQUESTED", 2),
      ],
    });
    expect(await gh.reader.pullRequest(context)).toEqual({
      context,
      mergeable: "CONFLICTING",
      mergeStateStatus: "DIRTY",
      reviewDecision: "CHANGES_REQUESTED",
      headRefOid: "c3",
      headRefName: "feature",
      baseRefName: "main",
      state: "OPEN",
      mergedAt: null,
      isDraft: false,
    });
    expect(gh.calls).toEqual([
      `${api}/pulls/42`,
      `${api}/pulls/42/reviews?per_page=100&page=1`,
    ]);
  });

  it("reads every page of reviews before deciding", async () => {
    const approvals = Array.from({ length: 100 }, (_, index) =>
      restReview(`reviewer-${index}`, "APPROVED", index + 1)
    );
    const gh = fakeGh({
      [`${api}/pulls/42`]: restPull(),
      [`${api}/pulls/42/reviews?per_page=100&page=1`]: approvals,
      [`${api}/pulls/42/reviews?per_page=100&page=2`]: [
        restReview("reviewer-0", "CHANGES_REQUESTED", 101),
      ],
    });
    expect((await gh.reader.pullRequest(context)).reviewDecision).toBe(
      "CHANGES_REQUESTED"
    );
    expect(gh.calls).toEqual([
      `${api}/pulls/42`,
      `${api}/pulls/42/reviews?per_page=100&page=1`,
      `${api}/pulls/42/reviews?per_page=100&page=2`,
    ]);
  });

  it("walks pages by number until a short page", async () => {
    const page = (start: number, count: number) =>
      Array.from({ length: count }, (_, index) => ({
        number: start + index,
        head: { ref: `branch-${start + index}` },
        base: { ref: "main" },
      }));
    const gh = fakeGh({
      [`${api}/pulls?state=open&per_page=100&page=1`]: page(1, 100),
      [`${api}/pulls?state=open&per_page=100&page=2`]: page(101, 100),
      [`${api}/pulls?state=open&per_page=100&page=3`]: page(201, 5),
    });
    const open = await gh.reader.openPullRequests(context);
    expect(open.length).toBe(205);
    expect([open[0], open[204]]).toEqual([
      { number: parsePrNumber(1), headRefName: "branch-1", baseRefName: "main" },
      {
        number: parsePrNumber(205),
        headRefName: "branch-205",
        baseRefName: "main",
      },
    ]);
    expect(gh.calls).toEqual([
      `${api}/pulls?state=open&per_page=100&page=1`,
      `${api}/pulls?state=open&per_page=100&page=2`,
      `${api}/pulls?state=open&per_page=100&page=3`,
    ]);
  });

  it("fails loudly when pages never run short", async () => {
    const full = Array.from({ length: 100 }, (_, index) => ({
      number: index + 1,
      head: { ref: `b${index}` },
      base: { ref: "main" },
    }));
    const gh = fakeGh(() => full);
    expect(await failureOf(() => gh.reader.openPullRequests(context))).toEqual(
      {
        kind: "missing-key",
        retryable: true,
        detail: "repos/owner/repo/pulls?state=open: more than 50 full pages",
      }
    );
    expect(gh.calls.length).toBe(50);
    expect(gh.calls[49]).toBe(`${api}/pulls?state=open&per_page=100&page=50`);
  });

  it("reads every check run and status of the head commit on the fast path", async () => {
    const gh = fakeGh({
      [`${api}/pulls/42`]: restPull(),
      [`${api}/commits/c3/check-runs?per_page=100&page=1`]: checkRuns([
        restCheckRun("build", "completed", "success"),
        restCheckRun("lint", "in_progress", null),
      ]),
      [`${api}/commits/c3/status?per_page=100&page=1`]: combinedStatus(
        "failure",
        [restStatus("ci/legacy", "failure")]
      ),
    });
    expect(await gh.reader.checksFastPath(context)).toEqual({
      kind: "checks",
      checks: [
        {
          kind: "passed",
          name: "build",
          reportedState: "SUCCESS",
          description: "",
          link: "https://github.com/owner/repo/actions/runs/1/job/build",
          workflow: "",
        },
        {
          kind: "pending",
          name: "lint",
          reportedState: "PENDING",
          description: "",
          link: "https://github.com/owner/repo/actions/runs/1/job/lint",
          workflow: "",
        },
        {
          kind: "failed",
          name: "ci/legacy",
          reportedState: "FAILURE",
          description: "ci/legacy is failure",
          link: "https://ci.example/ci/legacy",
          workflow: "",
        },
      ],
    });
    expect(gh.calls).toEqual([
      `${api}/pulls/42`,
      `${api}/commits/c3/check-runs?per_page=100&page=1`,
      `${api}/commits/c3/status?per_page=100&page=1`,
    ]);
  });

  it("reports the fast path unusable when a gh call exits non-zero", async () => {
    const gh = fakeGh({
      [`${api}/pulls/42`]: restPull(),
      [`${api}/commits/c3/check-runs?per_page=100&page=1`]: new Exit(
        1,
        "gh: Resource not accessible by integration (HTTP 403)\n"
      ),
    });
    expect(await gh.reader.checksFastPath(context)).toEqual({
      kind: "unusable",
      exitCode: 1,
      stderr: "gh: Resource not accessible by integration (HTTP 403)",
    });
  });

  it("pages the rollup fallback with page numbers as cursors", async () => {
    const full = Array.from({ length: 100 }, (_, index) =>
      restCheckRun(`job-${index}`, "completed", "success")
    );
    const gh = fakeGh({
      [`${api}/pulls/42`]: restPull(),
      [`${api}/commits/c3/check-runs?per_page=100&page=1`]: checkRuns(full),
      [`${api}/commits/c3/check-runs?per_page=100&page=2`]: checkRuns([
        restCheckRun("last", "completed", "failure"),
      ]),
      [`${api}/commits/c3/status?per_page=100&page=1`]: combinedStatus(
        "success",
        [restStatus("ci/legacy", "success")]
      ),
    });
    const first = await gh.reader.checkRollupPage(context, null);
    expect([first.checks.length, first.endCursor, first.checks[100]?.name]).toEqual(
      [101, "2", "ci/legacy"]
    );
    const second = await gh.reader.checkRollupPage(context, "2");
    expect([second.checks.map((check) => check.name), second.endCursor]).toEqual(
      [["last"], null]
    );
    expect(gh.calls).toEqual([
      `${api}/pulls/42`,
      `${api}/commits/c3/check-runs?per_page=100&page=1`,
      `${api}/commits/c3/status?per_page=100&page=1`,
      `${api}/pulls/42`,
      `${api}/commits/c3/check-runs?per_page=100&page=2`,
    ]);
    expect(
      await failureOf(() => gh.reader.checkRollupPage(context, "next"))
    ).toMatchObject({ detail: 'invalid check rollup cursor: "next"' });
  });

  it("fails closed through resolveChecks when the head commit has no checks", async () => {
    const gh = fakeGh({
      [`${api}/pulls/42`]: restPull(),
      [`${api}/commits/c3/check-runs?per_page=100&page=1`]: checkRuns([]),
      [`${api}/commits/c3/status?per_page=100&page=1`]: combinedStatus(
        "pending",
        []
      ),
    });
    expect(await failureOf(() => resolveChecks(gh.reader, context))).toEqual({
      kind: "checks-unavailable",
      retryable: true,
      detail: "could not read PR checks: fast path and paged rollup were empty",
    });
  });

  it("joins ccr review threads with REST review comments", async () => {
    const gh = fakeGh({
      [`${api}/pulls/42/ccr/review_threads`]: [
        { comment_ids: [11, 12], resolved: false, outdated: false },
        { comment_ids: [21], resolved: true, outdated: false },
      ],
      [`${api}/pulls/42/comments?per_page=100&page=1`]: [
        restComment(11, "octo", "please rename"),
        restComment(12, "boris", "done"),
        restComment(21, "octo", "typo"),
      ],
    });
    expect(await gh.reader.reviewThreads(context)).toEqual([
      {
        id: "11",
        firstComment: {
          authorLogin: "octo",
          body: "please rename",
          path: "a.ts",
          line: 3,
          createdAt: "2026-10-06T10:00:00Z",
        },
        isBugbot: false,
        bugbotReviewPasses: 0,
      },
    ]);
    expect(gh.calls).toEqual([
      `${api}/pulls/42/ccr/review_threads`,
      `${api}/pulls/42/comments?per_page=100&page=1`,
    ]);
  });

  it("rolls up the head and earlier commits newest first, stopping at the first pass", async () => {
    const gh = fakeGh({
      [`${api}/pulls/42/commits?per_page=100&page=1`]: ["c1", "c2", "c3", "c4"].map(
        (sha) => ({ sha, commit: { message: sha } })
      ),
      [`${api}/commits/c4/check-runs?per_page=100&page=1`]: checkRuns([
        restCheckRun("build", "completed", "failure"),
      ]),
      [`${api}/commits/c4/status?per_page=100&page=1`]: combinedStatus("pending", []),
      [`${api}/commits/c3/check-runs?per_page=100&page=1`]: checkRuns([
        restCheckRun("build", "in_progress", null),
      ]),
      [`${api}/commits/c3/status?per_page=100&page=1`]: combinedStatus("pending", []),
      [`${api}/commits/c2/check-runs?per_page=100&page=1`]: checkRuns([
        restCheckRun("build", "completed", "success"),
      ]),
      [`${api}/commits/c2/status?per_page=100&page=1`]: combinedStatus("pending", []),
    });
    expect(await gh.reader.commitRollups(context)).toEqual([
      { oid: "c4", state: "FAILURE" },
      { oid: "c3", state: "PENDING" },
      { oid: "c2", state: "SUCCESS" },
    ]);
    expect(gh.calls).toEqual([
      `${api}/pulls/42/commits?per_page=100&page=1`,
      `${api}/commits/c4/check-runs?per_page=100&page=1`,
      `${api}/commits/c4/status?per_page=100&page=1`,
      `${api}/commits/c3/check-runs?per_page=100&page=1`,
      `${api}/commits/c3/status?per_page=100&page=1`,
      `${api}/commits/c2/check-runs?per_page=100&page=1`,
      `${api}/commits/c2/status?per_page=100&page=1`,
    ]);
  });

  it("walks past a passing head to an earlier passing commit", async () => {
    const gh = fakeGh({
      [`${api}/pulls/42/commits?per_page=100&page=1`]: [{ sha: "c1" }, { sha: "c2" }],
      [`${api}/commits/c2/check-runs?per_page=100&page=1`]: checkRuns([
        restCheckRun("build", "completed", "success"),
      ]),
      [`${api}/commits/c2/status?per_page=100&page=1`]: combinedStatus("pending", []),
      [`${api}/commits/c1/check-runs?per_page=100&page=1`]: checkRuns([
        restCheckRun("build", "completed", "success"),
      ]),
      [`${api}/commits/c1/status?per_page=100&page=1`]: combinedStatus("pending", []),
    });
    expect(await gh.reader.commitRollups(context)).toEqual([
      { oid: "c2", state: "SUCCESS" },
      { oid: "c1", state: "SUCCESS" },
    ]);
  });

  it("reads at most ten commits before the head", async () => {
    const shas = Array.from({ length: 15 }, (_, index) => `c${index + 1}`);
    const gh = fakeGh((call) =>
      call.endsWith("/pulls/42/commits?per_page=100&page=1")
        ? shas.map((sha) => ({ sha }))
        : call.includes("/check-runs?")
          ? checkRuns([])
          : combinedStatus("pending", [])
    );
    expect(
      (await gh.reader.commitRollups(context)).map((rollup) => rollup.oid)
    ).toEqual(["c15", "c14", "c13", "c12", "c11", "c10", "c9", "c8", "c7", "c6", "c5"]);
  });

  it("stops after three earlier commits that ran CI without passing", async () => {
    const gh = fakeGh((call) =>
      call.endsWith("/pulls/42/commits?per_page=100&page=1")
        ? ["c1", "c2", "c3", "c4", "c5"].map((sha) => ({ sha }))
        : call.includes("/check-runs?")
          ? checkRuns([restCheckRun("build", "completed", "failure")])
          : combinedStatus("pending", [])
    );
    expect(
      (await gh.reader.commitRollups(context)).map((rollup) => rollup.oid)
    ).toEqual(["c5", "c4", "c3", "c2"]);
  });

  it("reads history once across polls and only the head after that", async () => {
    const commits = ["c1", "c2", "c3", "c4"];
    const routes: Record<string, unknown> = {
      [`${api}/pulls/42`]: restPull({ head: { ref: "feature", sha: "c4" } }),
      [`${api}/pulls/42/reviews?per_page=100&page=1`]: [],
      [`${api}/pulls/42/ccr/review_threads`]: [],
      [`${api}/pulls/42/comments?per_page=100&page=1`]: [],
      [`${api}/pulls/42/commits?per_page=100&page=1`]: commits.map((sha) => ({ sha })),
      [`${api}/commits/c4/check-runs?per_page=100&page=1`]: checkRuns([
        restCheckRun("build", "in_progress", null),
      ]),
      [`${api}/commits/c4/status?per_page=100&page=1`]: combinedStatus("pending", []),
      [`${api}/commits/c3/check-runs?per_page=100&page=1`]: checkRuns([]),
      [`${api}/commits/c2/check-runs?per_page=100&page=1`]: checkRuns([]),
      [`${api}/commits/c1/check-runs?per_page=100&page=1`]: checkRuns([
        restCheckRun("build", "completed", "success"),
      ]),
      [`${api}/commits/c1/status?per_page=100&page=1`]: combinedStatus("pending", []),
    };
    const gh = fakeGh(routes);
    const poll = async () => {
      const start = gh.calls.length;
      const snapshot = await readSnapshot({
        reader: gh.reader,
        context,
        pendingHistory: "include",
        allowDraft: false,
      });
      return {
        paths: gh.calls.slice(start).map((call) => call.slice(`${api}/`.length)),
        ci:
          snapshot.kind === "open"
            ? [snapshot.ci.kind, snapshot.ci.hadPreviousPassingCi]
            : snapshot.kind,
      };
    };
    const headPaths = (sha: string) => [
      "pulls/42",
      "pulls/42/reviews?per_page=100&page=1",
      "pulls/42/ccr/review_threads",
      "pulls/42/comments?per_page=100&page=1",
      `commits/${sha}/check-runs?per_page=100&page=1`,
      `commits/${sha}/status?per_page=100&page=1`,
    ];
    const first = await poll();
    expect(first).toEqual({
      paths: [
        ...headPaths("c4"),
        "pulls/42/commits?per_page=100&page=1",
        "commits/c3/check-runs?per_page=100&page=1",
        "commits/c2/check-runs?per_page=100&page=1",
        "commits/c1/check-runs?per_page=100&page=1",
        "commits/c1/status?per_page=100&page=1",
      ],
      ci: ["ci-pending", true],
    });
    expect(first.paths.length).toBe(11);
    const second = await poll();
    expect(second).toEqual({ paths: headPaths("c4"), ci: ["ci-pending", true] });
    expect(second.paths.length).toBe(6);

    routes[`${api}/pulls/42`] = restPull({ head: { ref: "feature", sha: "c5" } });
    routes[`${api}/pulls/42/commits?per_page=100&page=1`] = [...commits, "c5"].map(
      (sha) => ({ sha })
    );
    routes[`${api}/commits/c5/check-runs?per_page=100&page=1`] = checkRuns([
      restCheckRun("build", "in_progress", null),
    ]);
    routes[`${api}/commits/c5/status?per_page=100&page=1`] = combinedStatus("pending", []);
    expect(await poll()).toEqual({
      paths: [
        ...headPaths("c5"),
        "pulls/42/commits?per_page=100&page=1",
        "commits/c4/check-runs?per_page=100&page=1",
        "commits/c4/status?per_page=100&page=1",
      ],
      ci: ["ci-pending", true],
    });
  });

  it("finds the current PR by number or by the current branch", async () => {
    const remote = "git@github.com:owner/repo.git\n";
    const byNumber = fakeGh({
      "git remote get-url origin": remote,
      [`${api}/pulls/7`]: restPull({
        number: 7,
        html_url: "https://github.com/owner/repo/pull/7",
      }),
    });
    expect(await byNumber.reader.currentPr(parsePrNumber(7))).toEqual({
      owner: "owner",
      repo: "repo",
      number: parsePrNumber(7),
    });
    const byBranch = fakeGh({
      "git remote get-url origin": remote,
      "git branch --show-current": "feature/x\n",
      [`${api}/pulls?state=open&head=owner:feature%2Fx`]: [
        restPull({ number: 9, html_url: "https://github.com/owner/repo/pull/9" }),
      ],
    });
    expect(await byBranch.reader.currentPr(null)).toEqual({
      owner: "owner",
      repo: "repo",
      number: parsePrNumber(9),
    });
    expect(byBranch.calls).toEqual([
      "git remote get-url origin",
      "git branch --show-current",
      `${api}/pulls?state=open&head=owner:feature%2Fx`,
    ]);
    const none = fakeGh({
      "git remote get-url origin": remote,
      "git branch --show-current": "feature/x\n",
      [`${api}/pulls?state=open&head=owner:feature%2Fx`]: [],
    });
    expect(await failureOf(() => none.reader.currentPr(null))).toEqual({
      kind: "command-exit",
      retryable: true,
      code: 1,
      detail: 'no open pull requests found for branch "feature/x"',
    });
  });
});

describe("context and stack discovery", () => {
  it("returns a fully explicit context without any reader call", async () => {
    const reader = fakeReader();
    expect(
      await resolveContext({
        reader,
        owner: "explicit",
        repo: "repo",
        pr: context.number,
      })
    ).toEqual({ owner: "explicit", repo: "repo", number: context.number });
    expect(reader.calls).toEqual([]);
  });

  it("uses the local origin before currentPr for an explicit number", async () => {
    const reader = fakeReader({ origin: { owner: "local", repo: "checkout" } });
    expect(
      await resolveContext({
        reader,
        owner: null,
        repo: null,
        pr: context.number,
      })
    ).toEqual({ owner: "local", repo: "checkout", number: context.number });
    expect(reader.calls).toEqual(["originRepo"]);
  });

  it("orders the connected stack bottom-to-top", () => {
    const ordered = orderStack(context, [
      {
        number: parsePrNumber(41),
        headRefName: "base-feature",
        baseRefName: "main",
      },
      {
        number: context.number,
        headRefName: "feature",
        baseRefName: "base-feature",
      },
      {
        number: parsePrNumber(43),
        headRefName: "upstack",
        baseRefName: "feature",
      },
    ]);
    expect(ordered.map((item) => Number(item.number))).toEqual([41, 42, 43]);
  });
});
