## GO-CODE-QUALITY-GATE.md

> Second-pass revision: this document is intentionally expansive. It is a Go-native quality gate, not a mechanical Rust-to-Go translation. Rust-only concepts were replaced with Go module, package, context, goroutine, build tag, cgo, unsafe, testing, and vulnerability-management controls.

## Purpose

This document defines the Go language quality gate for this implementation phase.

Its purpose is to prevent low-quality Go code from being generated, accepted, copied into the project root, or treated as complete without measurable evidence.

This is not a style preference document. It is an engineering control document.

Go gives teams a small language, a strong standard library, fast builds, first-class testing, built-in formatting, modules, and a mature concurrency model. Go code can still be poor software. Go code can still be:

- architecturally wrong
- over-abstracted
- under-abstracted
- panic-prone
- nil-prone
- race-prone
- goroutine-leaking
- context/cancellation-unsafe
- under-tested
- unauditable
- dependency-heavy
- supply-chain risky
- cgo/unsafe-heavy without justification
- build-tag fragile
- semver-breaking
- hard to understand
- hard to maintain
- legally/regulatorily wrong
- business-incorrect even when it compiles

The implementation is not complete when files are created. The implementation is complete only when the code:

- builds
- is formatted
- passes vet/static analysis or documents justified exceptions
- has meaningful automated tests
- preserves architectural boundaries
- models errors explicitly
- avoids unnecessary panics
- avoids unsafe and cgo unless justified
- handles goroutines, contexts, and channels deliberately
- controls module, build-tag, and dependency sprawl
- is secure by default
- is auditable where required
- has measurable evidence in PHASE-RESULT.md

This document must be followed together with:

- AGENTS.md
- PHASE-PLAN*.md
- QUALITY-GATES.md
- LANGUAGE-QUALITY-GATE.md
- architecture.md
- myrules.txt

If this file conflicts with a phase-specific rule, follow the stricter rule unless the deviation is explicitly documented in PHASE-RESULT.md.

## 1. Non-Negotiable Completion Rule

The implementation LLM must not declare the phase complete merely because Go code was written.

A phase is complete only when:

1. The planned implementation exists.
2. Relevant automated tests exist.
3. The code builds.
4. Formatting was checked.
5. go vet and configured linters were executed where available.
6. The applicable quality gates were executed.
7. Failures were fixed or documented.
8. PHASE-RESULT.md was created.
9. The quality score is supported by evidence.

PHASE-RESULT.md must exist before the final message is sent.

The final message must be exactly:

I finished the implementation

No extra words. No summary. No apology. No markdown.


### Second-pass Go hardening

- Completion requires evidence from the exact repository state that will be handed off, not from a scratch directory, stale branch, or partial copy.
- Generated Go files count as implementation and must satisfy the same quality gate unless explicitly excluded with a reason.
- A successful `go test` alone is not enough when the phase changed dependencies, concurrency behavior, public API, serialization contracts, legal rules, security behavior, or persistence.
- Any skipped command must include the concrete blocker, not a vague statement such as "tool unavailable".
- Any command run from the wrong directory is invalid evidence.
- Any command run before the final code changes is stale evidence.
- Any manual test must be described as manual evidence and must not replace automated tests for business logic.
- PHASE-RESULT.md must explain residual risk in plain language.
- The implementation LLM must not inflate the quality score for code that lacks failure-path tests.
- The final response rule is part of the gate because premature summaries often hide missing evidence.

## 2. Go Toolchain Policy

## Recommendation

Use the Go toolchain defined by the project.

Go has stable releases and module-level language/toolchain declarations. Quality depends on a clear, explicit, documented toolchain policy.

For new Go projects, prefer:

- stable Go
- a documented go directive in go.mod
- a documented toolchain directive when the project intentionally pins or recommends a toolchain
- explicit GOOS, GOARCH, and CGO_ENABLED assumptions for deployable binaries
- boring, idiomatic Go over clever Go

## Always do

- Use the Go version already defined by the project.
- Use the go directive already defined in go.mod.
- Use the toolchain directive when present.
- Document the Go version used in PHASE-RESULT.md.
- Document relevant go env values in PHASE-RESULT.md.
- Keep go.mod language version consistent across the module/workspace unless intentionally mixed.
- Keep CI-equivalent commands runnable locally.
- Use stable Go unless the project explicitly requires development builds.
- Document any Go version or toolchain change.
- Verify that dependency additions do not silently require a higher Go version.
- Verify behavior under the intended GOOS/GOARCH/CGO_ENABLED combination when deployable artifacts are affected.

## Prefer

- The latest project-approved stable Go release.
- Explicit module and workspace policy.
- toolchain directive for reproducible team behavior when project-approved.
- go env evidence for important environment-dependent phases.
- Explicit migration notes when raising the go directive.
- Small, routine Go upgrades with tests rather than large toolchain jumps.
- Standard library features when they are clear, stable, and compatible with the project go directive.

## Avoid

- Relying on whatever Go version is installed locally.
- Raising the go directive without documenting impact.
- Using new standard-library APIs that violate the project’s Go version policy.
- Depending on GOWORK state accidentally.
- Depending on local GOPATH state.
- Depending on local module cache state.
- Changing CGO_ENABLED behavior without documenting runtime/deployment impact.
- Treating a successful local build as evidence for all target platforms.

## Almost never do

- Use development or unreleased Go builds in production business code without a documented architectural reason.
- Hide the required Go version only in local developer notes.
- Disable vet/static checks globally to finish a phase.
- Use toolchain changes as a substitute for clean design.
- Change GOOS/GOARCH/CGO_ENABLED assumptions inside a phase without documenting why.


### Second-pass Go hardening

- Treat the `go` directive as part of the language contract; it can affect available language semantics and standard-library APIs.
- Treat the `toolchain` directive as an operational choice; changing it can change developer and CI behavior.
- Document whether the phase was verified with `GOWORK=off` or with a checked-in workspace when multi-module behavior matters.
- Document whether `CGO_ENABLED` was relevant; cgo changes can alter linking, container images, cross-compilation, and deployment.
- Verify the target platform for binaries, serverless functions, containers, and CLIs instead of assuming the host platform matches production.
- Do not introduce code that requires a newer Go release unless raising the Go version is part of the planned change.
- When using recent language behavior, document that the module go directive supports it.
- For reusable libraries, consider downstream users before raising the Go version.
- For long-lived services, record the actual `go version` output, not only the intended version.
- For tool-managed code generation, record generator versions separately from the Go compiler version.

## 3. Go Modules and Build Reproducibility

## Recommendation

The build must be reproducible from a clean checkout using documented Go commands.

The implementation LLM must assume that isolated phase code will later be copied into the project root. Therefore, every phase must keep its build behavior predictable.

## Always do

- Use Go modules as the source of truth.
- Keep go.mod clean and minimal.
- Commit go.mod and go.sum for modules that build applications, services, CLIs, workers, libraries, and internal products.
- Keep dependency versions intentional.
- Keep module graph changes small.
- Run go mod tidy when dependencies change.
- Use -mod=readonly in verification when reproducibility matters.
- Avoid local-only replace directives.
- Avoid machine-specific configuration.
- Avoid hidden dependency on GOWORK unless the workspace is part of the project policy.
- Make build commands documented in PHASE-RESULT.md.
- Keep build behavior independent from IDE state.
- Keep build behavior independent from uncommitted local files.

## Prefer

- One module per coherent deployable or reusable boundary.
- A workspace only when the project intentionally manages multiple modules together.
- Minimal dependency graph.
- go list checks when dependency changes are non-trivial.
- Explicit package boundaries for architecture.
- Internal packages for implementation details.
- cmd/<name> packages for binaries.
- Reproducible generate/build/test steps.
- Vendoring only when the project policy requires it.

## Avoid

- replace directives pointing to local paths that only work on one machine.
- Pseudo-versions without reason.
- Unnecessary indirect dependencies.
- Dependencies added for trivial helpers.
- Multiple modules that drift without a workspace policy.
- Build behavior hidden inside go generate without documentation.
- Code generation that is not deterministic.
- Generated code that changes every run because of timestamps, local paths, or nondeterministic ordering.

## Almost never do

- Add dependencies just because they are convenient.
- Use go generate to hide business logic.
- Fetch remote resources during normal generation/build without explicit approval.
- Delete go.sum to make dependency resolution pass.
- Make build success depend on uncommitted local files.
- Use a replace directive as an architecture escape hatch.


### Second-pass Go hardening

- Reproducibility means a clean clone can run the documented commands without relying on local module cache, local replace paths, hidden environment variables, or IDE actions.
- `go.work` is allowed only when the repository intentionally uses it; accidental workspace state must not be required for builds.
- If the project uses vendoring, document whether verification used `-mod=vendor` and whether vendor contents were refreshed intentionally.
- If private modules are used, document the required `GOPRIVATE`/`GONOSUMDB` policy without exposing credentials.
- Check that package imports use the intended module path, not temporary paths created during generation.
- Ensure generated imports do not point to local scratch modules.
- Verify that command packages under `cmd/` still build after library/package refactors.
- Verify that test-only dependencies do not leak into production packages.
- Keep `replace` and `exclude` directives explainable in PHASE-RESULT.md.
- Avoid adding a second module merely to avoid fixing package boundaries.

## 4. go.mod and go.sum Policy

## Recommendation

Go modules use go.mod for declared requirements and go.sum for module checksums. Both are quality artifacts.

## Always do

- Commit go.mod.
- Commit go.sum when it exists.
- Document go.mod changes when dependencies, Go version, toolchain, module path, replace, exclude, or retract directives are modified.
- Run tests after dependency graph changes.
- Run vulnerability checks after meaningful dependency graph changes.
- Keep replace directives temporary and documented unless they are intentional project policy.
- Verify no accidental dependency remains after go mod tidy.

## Prefer

- go mod tidy with no resulting diff after committed dependency changes.
- Targeted go get module@version updates.
- Small, intentional go.mod/go.sum diffs.
- Reviewing transitive dependency changes with go list -m all.
- Checking why dependencies are needed with go mod why.
- Pinning tool dependencies through tools.go or project-approved tool management.
- Keeping tool dependencies separate from production dependencies when possible.

## Avoid

- Large accidental go.sum churn.
- Blind go get -u ./...
- Committing dependency changes unrelated to the phase.
- Ignoring retracted, incompatible, or vulnerable versions.
- Assuming a successful go mod tidy means the dependency graph is safe.
- Leaving local replace directives in committed code without explicit reason.

## Almost never do

- Delete go.sum from a real module to make CI pass.
- Manually edit go.sum.
- Accept a critical advisory because “it is transitive.”
- Let dependency resolution changes go undocumented in critical phases.
- Use GOPRIVATE/GONOSUMDB settings to bypass integrity checks without project policy.


### Second-pass Go hardening

- `go.mod` changes must be reviewed as source changes, not treated as tool noise.
- `go.sum` changes must be explained when they are large, unexpected, or unrelated to an intentional dependency change.
- After `go mod tidy`, inspect whether any dependency was removed because tests or build tags were not exercised.
- Run relevant tagged tests before deciding that a dependency is unused.
- Use `go list -m -u all` only as inspection; do not blindly upgrade everything inside a phase.
- Review retractions and major-version changes when dependencies are updated.
- Do not commit local module paths in `replace` unless the repository intentionally has a multi-module layout requiring them.
- Do not bypass checksum verification except under an explicit private-module policy.
- If a dependency is replaced by a fork, document the owner, commit/version, reason, and exit plan.
- If the module path changes, document downstream import impact and semver implications.

## 5. Mandatory Command Evidence

The implementation LLM must run the applicable commands and document the result in PHASE-RESULT.md.

If a command cannot be run, the reason must be documented.

## Baseline commands

- _**`go version`**_
- _**`go env GOVERSION GOOS GOARCH CGO_ENABLED GOMOD GOWORK GOPROXY GOSUMDB GOPRIVATE`**_
- _**`test -z "$(gofmt -l .)"`**_
- _**`go test ./...`**_
- _**`go vet ./...`**_
- _**`go test -race ./...`**_ where race detector is supported and practical
- _**`go mod tidy`**_ followed by verification that no unintended diff remains

## Stronger baseline for applications/services

- _**`go test -count=1 ./...`**_
- _**`go test -count=1 -race ./...`**_
- _**`go test -covermode=atomic -coverprofile=coverage.out ./...`**_
- _**`go tool cover -func=coverage.out`**_
- _**`go vet ./...`**_
- _**`staticcheck ./...`**_ when available
- _**`golangci-lint run ./...`**_ when configured
- _**`govulncheck ./...`**_ when available
- _**`go build ./...`**_ when packages include binaries or build-only paths

## Build-tag, cgo, and platform compatibility commands when applicable

- _**`CGO_ENABLED=0 go test ./...`**_
- _**`CGO_ENABLED=1 go test ./...`**_
- _**`GOOS=linux GOARCH=amd64 go test ./...`**_ or the project target
- _**`go test -tags="tag1 tag2" ./...`**_
- _**`go test -run TestIntegration -tags=integration ./...`**_ when integration tests are gated by tags
- _**`go list -deps ./...`**_
- _**`go mod why -m <module>`**_ for non-trivial new dependencies

## Optional but recommended when configured

- _**`go test -fuzz=<FuzzName> -fuzztime=60s ./...`**_
- _**`go test -bench=. -benchmem ./...`**_ for performance-sensitive code
- _**`go test -run TestName -count=100 ./...`**_ for suspected flakes
- _**`go test -shuffle=on ./...`**_
- _**`go test -timeout=60s ./...`**_ or project timeout
- _**`go test -json ./...`**_ for machine-readable evidence
- _**`go tool pprof`**_ evidence for performance/resource phases
- _**`gremlins`**_ or project-approved mutation tooling for critical rules
- _**`gosec ./...`**_ when configured
- _**`go-licenses`**_ or project-approved license checks when configured
- _**`go doc`**_ or documentation generation checks for public packages

## Required evidence format

PHASE-RESULT.md must include:

## Commands run

- _**`command here`**_

## Commands passed

- _**`command here`**_

## Commands failed

- _**`command here`**_
- Reason:
- Impact:
- Required fix:

## Recommendation

A command that was not run is not evidence.

A command that failed but was ignored is negative evidence.

A command that failed because a tool is unavailable must still be documented.


### Second-pass Go hardening

- For packages with build tags, run at least the default tag set and every tag set affected by the phase.
- For code that uses `context`, goroutines, channels, locks, maps shared across goroutines, timers, or background workers, race/cancellation evidence is required unless impossible on the platform.
- For code that touches public API, add documentation/API compatibility evidence where relevant.
- For code that adds dependencies, run vulnerability and dependency inspection commands when available.
- For code that handles untrusted input, run fuzz/property tests when useful and static/security tools when configured.
- For code that changes generated artifacts, run generation and verify the diff is deterministic.
- For code that changes serialization, include tests for both encoding and decoding.
- For code that changes external integration behavior, include fake-server/contract-test evidence.
- For code that changes persistence, include migration/query/mapping test evidence.
- For legal, audit, signing, and eSocial/SST code, include golden/contract/error/audit tests and do not rely only on broad package tests.

## Additional command examples

- _**`go test -run=^$ ./...`**_ to compile tests quickly when a full test run is blocked, with the blocker documented.
- _**`go test -run TestName -count=20 ./path`**_ for timing-sensitive or previously flaky tests.
- _**`go test -race -run TestConcurrentWorkflow ./path`**_ for targeted concurrency evidence.
- _**`go test -coverpkg=./... -coverprofile=coverage.out ./...`**_ when cross-package coverage is meaningful.
- _**`go test -tags='integration' ./...`**_ when integration behavior is behind build tags.
- _**`GOWORK=off go test ./...`**_ when ensuring a module does not accidentally depend on local workspace state.
- _**`go list -deps -test ./...`**_ when reviewing test and production dependency graphs.
- _**`go test -json ./... > test-results.json`**_ when PHASE-RESULT.md references machine-readable test evidence.

## 6. Formatting and Style

## Recommendation

Use gofmt. Do not debate formatting. Automate it.

## Always do

- Run gofmt checks.
- Keep formatting deterministic.
- Keep imports clean.
- Use gofmt/goimports according to project policy.
- Use idiomatic Go naming.
- Keep package names short, lowercase, and meaningful.
- Keep exported names clear and documented where required.
- Remove unused imports.
- Avoid formatting churn unrelated to the phase.

## Prefer

- Default gofmt formatting.
- goimports when import grouping/cleanup is project standard.
- Small files organized around cohesive responsibilities.
- Consistent ordering inside files:
  - package documentation
  - package declaration
  - imports
  - constants/variables
  - public types
  - private types
  - constructors
  - methods/functions
  - compile-time interface assertions when useful
- Short functions with clear control flow.

## Avoid

- Manual formatting fights against gofmt.
- Clever alignment.
- Long functions hidden by formatting.
- Reformatting files unrelated to the phase.
- Package names such as common, utils, helpers, misc, or shared without strong reason.
- Exporting names only because tests or other packages were poorly placed.

## Almost never do

- Disable formatting checks.
- Leave code unformatted.
- Commit generated formatting churn without reason.
- Use formatting to hide overly complex code.


### Second-pass Go hardening

- `gofmt` is mandatory; `goimports` is recommended when import grouping or unused imports are likely.
- Formatting evidence should verify no files remain unformatted, not merely run `gofmt` silently.
- Do not mix multiple packages in the same directory except the normal `_test` external package pattern.
- Keep package comments and exported identifiers formatted for `godoc` readability.
- Keep table tests readable after formatting; if the table is unreadable, the test design probably needs simplification.
- Avoid overlong composite literals that hide missing fields or mistaken field order.
- Prefer keyed struct literals for exported or non-trivial structs.
- Keep generated file headers stable and clear.
- Do not reformat unrelated files unless the phase explicitly includes formatting cleanup.
- Style is not a substitute for architecture, but bad formatting is still a gate failure.

## 7. Vet, Static Analysis, and Lint Policy

## Recommendation

go vet and configured linters must be treated as quality gates, not suggestions.

Use static analysis to detect suspicious code, incorrect printf formats, lost cancellations, nil risks, shadowing mistakes, unnecessary complexity, poor idioms, and maintainability issues.

## Always do

- Run go vet.
- Run project-configured linters where available.
- Treat new warnings as failures unless explicitly justified.
- Fix linter findings unless there is a documented reason.
- Keep suppressions narrow.
- Add a reason when suppressing a lint.
- Avoid blanket exclusions.
- Document new suppressions in PHASE-RESULT.md.

## Prefer

- staticcheck for deeper static analysis.
- golangci-lint as a configured linter runner when project-approved.
- Linters for:
  - errcheck
  - govet
  - staticcheck
  - ineffassign
  - unused
  - revive or stylecheck when project-approved
  - contextcheck where useful
  - bodyclose for HTTP response handling
  - gosec for security-sensitive code
  - depguard/import restrictions for architecture boundaries
- CI-equivalent lint commands.
- Per-line suppressions with explicit justification.

## Avoid

- Suppressing warnings to avoid refactoring.
- Broad nolint comments.
- Ignoring unchecked errors.
- Ignoring lost cancel functions.
- Ignoring suspicious goroutine/channel findings.
- Leaving fmt.Println/log.Printf debug statements in production code unless policy allows it.
- Adding linter exclusions without comments.

## Almost never do

- Disable go vet for critical packages.
- Disable all linters to finish a phase.
- Suppress panic, security, race, or unchecked-error findings in eSocial/SST, security, signing, persistence, or audit code.
- Treat linter output as cosmetic when it identifies real risk.


### Second-pass Go hardening

- `go vet` is baseline evidence; stronger projects should also use `staticcheck` or configured `golangci-lint`.
- Treat static analysis findings as correctness signals unless the finding is demonstrably false positive.
- Suppressions must be local, narrow, and commented with a reason.
- Do not add `//nolint` to make generated code pass unless the generator cannot be changed and the risk is documented.
- Review unchecked errors with `errcheck` or equivalent where configured.
- Review shadowing and nilness findings carefully in business, persistence, and serialization code.
- Review copy-lock findings; copying a value containing a mutex can break synchronization.
- Review loop-variable capture findings, especially in modules using older Go semantics or mixed go directives.
- Review printf-format findings because they often hide broken error messages or logging fields.
- `staticcheck`/lint unavailability is not a pass; it must be documented as unavailable.

## 8. Naming Rules

## Recommendation

Names must reveal intent.

A maintainer should understand the purpose of a type, interface, function, package, or test without reading its full implementation.

## Always do

- Use domain language.
- Use precise names.
- Name functions by behavior.
- Name packages by responsibility.
- Name tests by expected behavior.
- Use Go naming conventions.
- Keep abbreviations only when domain-standard.
- Use typed aliases or defined types for important domain concepts.
- Use names that distinguish raw data from validated data.

## Prefer

- EmployeeExposurePeriod over PeriodData.
- ESocialEventBatch over Batch.
- SignedXMLDocument over XMLResult.
- OccupationalRiskAssessment over RiskInfo.
- EventTransmissionReceipt over ResponseData.
- ValidateEventVersion over ProcessVersion.
- CanBeCancelled over CheckStatus.
- RawEventPayload for untrusted input.
- ValidatedEventPayload after validation.
- UnsignedXMLDocument before signing.
- SignedXMLDocument after signing.
- ID, URL, XML, JSON, API capitalization consistent with Go initialism conventions.

## Avoid

- Names such as:
  - helper
  - utils
  - common
  - processor
  - manager
  - handler
  - data
  - stuff
- Technical-only names for domain concepts.
- Ambiguous interfaces such as Processor or Manager.
- Overloaded terms with multiple meanings.
- Boolean parameters whose meaning is unclear.
- Naming domain packages after frameworks.

## Almost never do

- Use placeholder names in production code.
- Use single-letter names outside tiny local scopes.
- Name domain types only after database tables or transport payloads.
- Let generated names define the domain language.
- Use Thing, Object, Item, or Entry when a domain name exists.


### Second-pass Go hardening

- Package names should be short, lowercase, and meaningful; avoid `utils`, `helpers`, `common`, and `models` dumping grounds.
- Avoid stutter: `payment.Invoice`, not `payment.PaymentInvoice`, when the package name already supplies context.
- Preserve common initialisms consistently: `ID`, `URL`, `HTTP`, `XML`, `JSON`, `API`, `SQL`, `CPF`, `CNPJ` when domain-standard.
- Use domain names for business concepts rather than transport, database, or framework names.
- Distinguish raw input from validated domain values in names.
- Name tests as behavior statements, such as `TestValidateEventRejectsExpiredCertificate`.
- Avoid names that encode implementation details when the concept is domain-level.
- Avoid vague interfaces such as `Manager`, `Handler`, `Processor`, `Service` unless the domain itself uses that term.
- Avoid booleans with unclear meaning; prefer named options or domain-specific methods.
- Do not let generated payload names become the vocabulary for business rules without review.

## 9. Package and Module Structure

## Recommendation

Go packages and modules must reflect architecture, not random grouping.

A good Go structure makes invalid dependencies visible and hard to introduce.

## Always do

- Preserve architecture.md.
- Keep domain logic separate from infrastructure.
- Keep application/use-case orchestration separate from adapters.
- Keep API/transport models separate from domain models.
- Keep external clients outside the domain.
- Keep package visibility narrow through unexported names.
- Use internal/ packages when implementation must not be imported outside a boundary.
- Keep public package surface small.
- Avoid exporting internals accidentally.
- Keep package cycles out of the design.

## Prefer

For a single-module service:

```text
cmd/
  service/
    main.go
internal/
  domain/
    model/
    rules/
    events/
    errors/
  application/
    usecase/
    ports/
    commands/
    results/
  infrastructure/
    persistence/
    esocial/
    xml/
    signing/
    messaging/
  api/
    http/
    dto/
    mapper/
  config/
pkg/
  public reusable packages only when intentionally public
```

For a larger workspace:

```text
modules or packages for:
  domain
  application
  infrastructure
  api
  app/cmd
```

## Avoid

- One huge main.go.
- One giant package with unrelated responsibilities.
- One giant internal/common package.
- Domain packages importing infrastructure packages.
- API packages containing business rules.
- Infrastructure returning provider-specific models into the domain.
- Cyclic conceptual dependencies.
- Exporting everything because tests were placed outside the package.
- Creating a module for every tiny concept.

## Almost never do

- Put business rules in:
  - HTTP handlers
  - database adapters
  - XML builders
  - generated payload structs
  - message consumers
  - external API clients
  - CLI argument parsing
- Create a common package that becomes a dumping ground.
- Use exported names as an afterthought.
- Let package structure be dictated by a framework generator.


### Second-pass Go hardening

- Go package boundaries are architectural boundaries; imports must show allowed dependency direction.
- Use `internal/` to prevent accidental external imports of implementation packages.
- Use `cmd/<binary>` for composition and process entrypoints; keep business rules out of `main`.
- Avoid a large root package that contains domain, persistence, HTTP, XML, and configuration together.
- Keep package dependency cycles impossible by design, not merely absent today.
- Prefer package names based on behavior/responsibility, not layer buzzwords alone.
- Keep generated DTO packages separate from domain packages when invariants differ.
- Keep adapter packages provider-specific when integration behavior differs.
- Do not use a top-level `pkg/` directory as a substitute for deciding public API boundaries.
- Public packages must be intentionally importable and supported.

## Example package layout for a service

```text
cmd/
  app/
    main.go
internal/
  domain/
    employee/
    esocial/
    audit/
  application/
    send_event/
    cancel_event/
    ports/
  infrastructure/
    postgres/
    signer/
    esocialclient/
    clock/
  transport/
    httpapi/
    queue/
  config/
```

This is a pattern, not a mandate. The project architecture wins, but packages must make invalid dependencies difficult.

## 10. Architectural Boundaries

## Recommendation

Business rules must be explicit, isolated, and tested.

Go packages, internal visibility, interfaces, and dependency direction should enforce boundaries, not bypass them.

## Always do

- Keep dependency direction inward.
- Put business rules in domain/application packages.
- Put side effects in infrastructure/adapters.
- Use interfaces/ports at boundaries where useful.
- Keep handlers thin.
- Keep persistence code focused on persistence.
- Keep external clients focused on communication.
- Keep XML/JSON generation separate from business decisions.
- Test boundary rules.
- Make boundary crossing explicit with mappers.
- Keep framework-specific types out of the domain.

## Prefer

- Domain types with invariants.
- Application services/use cases for orchestration.
- Interfaces for persistence, clocks, signing, storage, external APIs, messaging.
- Adapter implementations outside the core.
- DTO-to-domain mappers at boundaries.
- Domain errors separate from infrastructure errors.
- internal/ packages for hard boundaries.
- Import restriction checks when package boundaries are critical.

## Avoid

- Domain depending on SQL, HTTP, XML, JSON, logging, or framework-specific APIs.
- Handlers calling database code directly for business workflows.
- Infrastructure deciding domain outcomes.
- API DTOs reused as domain objects.
- Database records reused as API responses.
- Provider payloads leaking into core logic.
- Business rules hidden in struct tags.
- Letting context.Context leak into pure domain logic unless cancellation/deadline is truly part of the abstraction.
- Letting database transaction types leak into domain logic.

## Almost never do

- Hide business decisions inside SQL queries.
- Hide legal rules inside XML serialization.
- Hide validation inside transport mappers only.
- Change architecture inside a phase without documenting the reason.
- Make domain correctness depend on a framework.
- Put audit/legal decisions in logging side effects.


### Second-pass Go hardening

- Domain packages must not import HTTP routers, SQL drivers, logging frameworks, queue clients, or generated external payload packages unless the architecture explicitly allows it.
- Application/use-case packages may coordinate ports, transactions, clocks, and domain rules, but should not hide provider-specific behavior.
- Transport handlers should parse, validate boundary shape, call use cases, map results, and return responses.
- Persistence adapters should map database records to domain/application types explicitly.
- XML/JSON builders should not decide whether a legal event is valid; they should serialize already validated state.
- Audit decisions should be part of application/domain flow, not incidental log statements.
- Boundary mappers must be tested because they are frequent sources of business bugs.
- Do not pass `context.Context` into pure domain logic unless cancellation/deadline is genuinely part of that domain operation.
- Do not let framework middleware become the only place where authorization or tenancy rules live if they affect domain correctness.
- If architecture changes are necessary, document the reason, alternatives considered, and residual risk in PHASE-RESULT.md.

## 11. Go Type System as a Quality Tool

## Recommendation

Use Go’s type system to make invalid states hard to represent.

Do not use strings, booleans, maps, or raw primitives when a meaningful domain type is needed.

## Always do

- Use defined types for important domain identifiers and constrained values.
- Use constants with typed enums for closed sets of states.
- Use structs for meaningful grouped data.
- Keep invariants enforced at construction.
- Prefer compile-time guarantees over runtime comments.
- Avoid exposing constructors or fields that allow invalid state.
- Distinguish untrusted, validated, signed, sent, rejected, and persisted states in types where useful.
- Use parsing/validation functions for fallible construction.

## Prefer

- type EmployeeID string instead of raw string.
- type EventVersion string with validation.
- type SignedXMLDocument []byte or struct wrapper instead of raw []byte.
- type TransmissionProtocol string instead of raw protocol string.
- NonEmptyString-style value objects where important.
- DateRange instead of two unrelated dates.
- Enum-like state machines for legal event lifecycle.
- Validated[T] wrappers only when they clarify state.
- Constructors returning (T, error) for invalid inputs.

## Avoid

- map[string]string as a domain model.
- map[string]any as a domain model.
- interface{} or any as business data.
- Boolean flags that change behavior.
- Pointers for required fields just to express optionality.
- Primitive obsession for important concepts.
- Invalid intermediate states.
- Domain code that accepts raw transport payloads.
- Business state represented by magic strings.

## Almost never do

- Represent legal/event state as arbitrary strings.
- Represent money, dates, measurements, certificates, event IDs, or legal codes as unvalidated raw primitives in domain code.
- Use comments to describe invariants that types could enforce.
- Use generic maps for eSocial/SST event structures.


### Second-pass Go hardening

- Go does not have Rust-style exhaustive enums; compensate with constructors, validation, tests, and default-case discipline.
- Use custom types for identifiers, statuses, versions, codes, and constrained values when raw strings/ints would be ambiguous.
- Keep fields unexported when invariants must be protected.
- Provide constructors returning `(T, error)` for validated values.
- Use typed constants for closed sets, but validate external strings before converting them.
- Use methods to expose behavior instead of exposing mutable fields.
- Use small structs to group values that must stay consistent, such as date ranges or signed payload metadata.
- Avoid `map[string]any` as a domain model.
- Avoid `any` and reflection in domain code unless there is a documented reason.
- Use compile-time interface assertions only for adapters and boundaries where they add clarity.

## Example expectation

```go
type EventStatus string

const (
    EventStatusDraft    EventStatus = "draft"
    EventStatusSigned   EventStatus = "signed"
    EventStatusSent     EventStatus = "sent"
    EventStatusRejected EventStatus = "rejected"
)

func ParseEventStatus(raw string) (EventStatus, error) {
    switch EventStatus(raw) {
    case EventStatusDraft, EventStatusSigned, EventStatusSent, EventStatusRejected:
        return EventStatus(raw), nil
    default:
        return "", fmt.Errorf("invalid event status %q", raw)
    }
}
```

## 12. Pointers, Values, and Nil

## Recommendation

Pointer and value semantics should make code safer and simpler, not more fragile.

Avoid pointer-heavy APIs that create nil ambiguity and hidden mutation.

## Always do

- Choose pointer vs value deliberately.
- Use values for small immutable domain objects where practical.
- Use pointers when identity, mutation, large copies, or optionality truly matter.
- Keep nil behavior explicit.
- Avoid returning nil interfaces.
- Avoid nil maps/slices ambiguity in public contracts.
- Clone or copy slices/maps before storing when callers must not mutate internal state.
- Make mutation obvious in method receivers.
- Avoid sharing mutable data across goroutines without synchronization.

## Prefer

- Immutable value objects.
- Constructors that return concrete values and errors.
- Defensive copies for slices, maps, and byte buffers at boundaries.
- Explicit Optional-like domain modeling when absence has business meaning.
- Empty slices over nil slices for JSON/API responses when contract expects arrays.
- Value receivers for immutable small types.
- Pointer receivers for mutation or non-trivial copy avoidance.

## Avoid

- Pointer fields everywhere because generated structs did it.
- Pointers for required values.
- Nil maps that later panic on assignment.
- Mutating slices/maps received from callers without ownership clarity.
- Returning internal slices/maps directly.
- Hidden shared state through package variables.
- Copying structs that contain mutexes.

## Almost never do

- Use unsafe pointers to bypass design issues.
- Use nil to hide a technical failure.
- Allow domain objects to be invalid between setter calls.
- Depend on finalizers for business-critical cleanup.
- Rely on nil interface behavior without tests.


### Second-pass Go hardening

- Choose pointer receivers for mutation, large structs, synchronization-bearing structs, or identity-bearing types.
- Choose value receivers for small immutable value objects when copying is safe.
- Never copy a value containing `sync.Mutex`, `sync.RWMutex`, `sync.WaitGroup`, `atomic` fields, or other synchronization state.
- Avoid pointer-to-interface; use an interface value instead.
- Understand the nil-interface trap: an interface containing a typed nil pointer is not itself nil.
- Avoid returning pointers to mutable internal slices, maps, or structs unless mutation is intentional.
- Copy slices and maps at boundaries when callers must not mutate internal state.
- Avoid storing pointers to loop variables in code that must support older module semantics or unclear tooling assumptions.
- Be explicit about nil acceptance in public functions.
- Test nil inputs when nil is accepted or possible from external boundaries.

## 13. Immutability and Mutation

## Recommendation

Prefer immutable data and explicit state transitions.

Mutation must preserve invariants.

## Always do

- Keep fields unexported unless direct access is harmless and intentional.
- Make mutation explicit.
- Enforce invariants in constructors and mutation methods.
- Avoid exposing mutable internals.
- Prefer immutable value objects.
- Use methods to represent controlled mutation.
- Test state transitions.
- Keep mutation local and obvious.
- Do not let unrelated layers mutate domain state.

## Prefer

- Constructor/factory functions returning (T, error).
- Explicit methods such as:
  - MarkAsSigned
  - MarkAsSent
  - MarkAsRejected
  - Cancel
  - Correct
- Enum-based states.
- Builder pattern for complex construction, with final validation.
- Immutable command/result structs.
- State-specific types when transitions are critical:
  - DraftEvent
  - ValidatedEvent
  - SignedEvent
  - SentEvent
  - RejectedEvent

## Avoid

- Exported mutable fields in domain objects.
- Mutation through unrelated layers.
- Setters that allow invalid intermediate states.
- Global mutable maps/slices.
- sync.Mutex as a default design tool for domain state.
- Invalid temporary states.
- Builder APIs that allow required fields to be forgotten without validation.

## Almost never do

- Use global mutable state.
- Use init-time mutation to configure business behavior.
- Use sync.Once/package variables to hide poor lifecycle design.
- Allow domain objects to be invalid between setter calls.
- Depend on finalizers or goroutine cleanup for business state transitions.


### Second-pass Go hardening

- Go cannot make most data immutable by default; design APIs so mutation is controlled.
- Use unexported fields plus methods to preserve invariants.
- Avoid setter-heavy domain objects that can exist in invalid intermediate states.
- Return copies of slices/maps from domain objects unless callers are allowed to mutate internal state.
- Do not mutate input DTOs or caller-owned data unless the function contract says so.
- Keep state transitions explicit: `Sign`, `Send`, `Reject`, `Cancel`, `Correct`.
- Ensure transition methods validate current state and return errors for illegal transitions.
- Avoid global mutable registries and package-level mutable variables.
- Use `sync.Once` only for true one-time initialization, not as a hidden lifecycle controller.
- Test state transition matrices for critical legal/business workflows.

## 14. Error Handling

## Recommendation

Go errors must be explicit and meaningful.

Use error returns for recoverable failures. Use panic only for programmer bugs or truly impossible internal states.

## Always do

- Return errors for fallible operations.
- Define meaningful error values or types.
- Preserve source errors with wrapping.
- Include actionable context.
- Convert infrastructure errors at boundaries.
- Test failure paths.
- Distinguish business rejection from technical failure.
- Avoid leaking sensitive internals in external errors.
- Keep error semantics stable for important business behavior.
- Avoid losing root causes through string conversion.

## Prefer

- errors.Is and errors.As compatible error design.
- Sentinel errors only when callers need stable comparison.
- Custom error types for domain/application failures that need structured data.
- fmt.Errorf("context: %w", err) for wrapping.
- errors.Join only when multiple independent failures matter.
- User-safe messages plus internal traceability.
- Error codes for auditable/legal failures.
- Separate error types per layer:
  - domain error
  - application error
  - infrastructure error
  - API error

## Avoid

- Returning vague errors such as errors.New("failed").
- Stringly typed errors that callers parse.
- panic in production paths.
- Returning nil error with invalid/zero result.
- Returning nil result with nil error when absence is not valid.
- Collapsing all failures into one generic error.
- Losing root cause information.
- Treating business rejection as infrastructure failure.
- Treating infrastructure failure as business rejection.

## Almost never do

- Panic for recoverable business, validation, I/O, XML, signing, persistence, or integration failures.
- Swallow errors.
- Log and discard critical failures.
- Ignore returned errors.
- Treat eSocial/SST rejection as a generic technical error.
- Convert every error into a string at the boundary and lose structure.


### Second-pass Go hardening

- Return errors for recoverable failures; reserve panic for programmer mistakes and impossible internal invariants.
- Wrap errors with `%w` when callers need `errors.Is` or `errors.As`.
- Use typed errors or sentinel errors for stable business decisions.
- Do not compare error strings in production logic.
- Avoid returning `nil, nil` for lookups where absence has meaning; use `(T, bool, error)` or `ErrNotFound` deliberately.
- Preserve provider/root-cause errors inside infrastructure layers while mapping to safe application/API errors at boundaries.
- Do not log and return the same error at every layer; choose the layer that has useful context.
- Handle errors from deferred operations when they matter, especially file flush/close and transaction rollback/commit.
- Distinguish business rejection, validation failure, authorization failure, conflict, timeout, cancellation, and infrastructure failure.
- Test representative error branches, not only the happy path.

## Required error-review questions

- Can callers programmatically distinguish important outcomes?
- Is sensitive information hidden from external responses?
- Is root cause preserved for internal diagnostics?
- Are retryable and non-retryable failures distinguishable?
- Are context cancellation and deadline errors handled intentionally?

## 15. Panic, recover, TODO, and Fatal Policy

## Recommendation

Production Go must not rely on panics for normal control flow.

Panics are acceptable in tests, prototypes, and truly impossible internal invariants, but must be rare and justified.

## Always do

- Avoid panic in production code.
- Avoid log.Fatal/os.Exit in libraries and domain/application packages.
- Remove TODO stubs, panic("not implemented"), and debug prints before completion.
- Replace panics with typed errors when callers can recover.
- Document any intentional production panic in PHASE-RESULT.md.
- Use recover only at process/framework boundaries, never as business logic.
- Ensure panic recovery records useful observability and audit context when appropriate.

## Prefer

- Returning errors with context.
- Guard clauses for invalid input.
- Tests proving invariants instead of panics.
- Exhaustive switch behavior with default error handling when external state is involved.
- must-style helpers only in tests or package initialization for constants with clear invariants.

## Avoid

- panic after parsing external input.
- panic after XML/JSON serialization/deserialization.
- panic after signing, hashing, database, network, or filesystem operations.
- log.Fatal inside packages that are not main.
- recover that hides failures.
- Panicking while holding locks.
- Panicking in goroutines without supervision.

## Almost never do

- Use panic in eSocial/SST, legal, security, audit, signing, persistence, or integration code.
- Use recover as normal business control flow.
- Leave panic("TODO") in phase code and call it complete.
- Call os.Exit from reusable packages.
- Justify panic because “the AI knows this cannot happen.”


### Second-pass Go hardening

- `panic` must not be used for normal validation, I/O, persistence, signing, network, or integration failure.
- `log.Fatal` and `os.Exit` are process-level decisions and must stay in `main` or command composition code.
- Library, domain, and application packages must return errors instead of terminating the process.
- `recover` is not a business-control-flow tool; use it only at process/goroutine boundaries to prevent crash escalation when appropriate.
- Any recovered panic must be logged/observed and converted to a safe failure result.
- Do not let goroutine panics disappear; supervise goroutines and propagate failure.
- Remove `panic("TODO")`, `panic("not implemented")`, `TODO`, `FIXME`, `spew.Dump`, `fmt.Println`, and debugging leftovers before completion.
- In tests, `t.Fatal` is acceptable for setup failures, but assertions should still prove behavior.
- Panic tests must prove the panic is intentional and constrained.
- Document any intentional production panic in PHASE-RESULT.md.

## 16. Nil, Zero Values, and Absence Semantics

## Recommendation

Use nil and zero values only when their semantics are deliberate and tested.

Use errors when failure reason matters.

## Always do

- Use nil for legitimate absence only.
- Use errors for fallible operations.
- Convert missing required values into errors.
- Keep zero-value semantics clear.
- Test nil, zero, and error cases.
- Avoid ambiguous (*T, error) returns where nil/nil is possible.
- Avoid optional bool pointers when an enum clarifies meaning.

## Prefer

- Clear constructors that validate required values.
- Domain-specific absence states when absence has business meaning.
- Clear enum variants for stateful absence.
- Result patterns expressed through (T, error) or domain results.
- Typed validation errors.
- Returning (T, bool) only for map-like lookup semantics where absence is normal and no reason is needed.

## Avoid

- Returning nil for invalid input when caller needs a reason.
- Returning an error for ordinary lookup absence without policy.
- Using *bool for business state.
- Using *string when an enum would clarify meaning.
- Using any/interface{} to avoid modeling absence.
- Returning zero value and nil error for invalid states.

## Almost never do

- Use nil to hide a technical failure.
- Use panic for absence.
- Collapse validation failures into zero values.
- Return nil interface values accidentally.


### Second-pass Go hardening

- Use zero values only when they are valid and intentional for the domain concept.
- Use pointers in DTOs for optional JSON/XML fields when absence must be distinguished from zero.
- Avoid pointers in domain objects merely to express optionality when a clearer domain type exists.
- Distinguish nil slice, empty slice, and omitted field when API contracts care.
- Distinguish nil map from empty map; writing to a nil map panics.
- Use `(value, ok)` for ordinary map lookup absence.
- Use `error` when absence is exceptional or needs an explanation.
- Use explicit enums/status values for meaningful absence states.
- Validate required fields before constructing domain objects.
- Test absent, zero, empty, and invalid values separately when the contract distinguishes them.

## 17. unsafe Package Gate

## Recommendation

The unsafe package is forbidden by default unless the phase explicitly requires it.

If unsafe is required, it must be isolated, documented, tested, and justified.

## Always do

- Avoid unsafe unless there is a documented need.
- Keep unsafe code as small as possible.
- Provide a safety comment for every unsafe use.
- Explain the invariant that makes the unsafe operation valid.
- Expose safe abstractions over unsafe internals.
- Add tests around unsafe boundaries.
- Run race/fuzz tests where unsafe touches parsing, memory, or concurrency.
- Document unsafe justification in PHASE-RESULT.md.
- Document why safe Go was insufficient.
- Keep unsafe out of domain/business logic.

## Prefer

- Safe Go alternatives.
- Existing well-maintained packages with audited unsafe internals when necessary.
- Small private packages for unsafe internals.
- Fuzz/property tests for unsafe parsers or boundary code.
- Explicit invariants in type design.
- Build tags to isolate platform-specific unsafe code.
- Benchmarks proving unsafe is justified when used for performance.

## Avoid

- unsafe to bypass type safety.
- unsafe for micro-optimizations without measurement.
- unsafe in domain/business logic.
- unsafe in legal/eSocial/SST logic.
- Large unsafe functions.
- Unsafe public APIs without clear contracts.
- unsafe hidden in generated code.

## Almost never do

- Use unsafe.Pointer arithmetic in business applications.
- Use unsafe to share mutable state across goroutines.
- Introduce unsafe without tests and documentation.
- Claim high quality while unsafe invariants are undocumented.
- Use unsafe in generated code without explicit review.


### Second-pass Go hardening

- The `unsafe` package is forbidden by default.
- Any `unsafe` usage must be isolated in the smallest possible file/package.
- Every unsafe operation must have a nearby comment explaining the invariant that makes it safe.
- Expose a safe API over unsafe internals.
- Add tests for boundary cases, alignment assumptions, sizes, lifetimes, and invalid inputs.
- Add fuzz tests for unsafe parsing or conversion when applicable.
- Verify architecture/platform assumptions for unsafe code; pointer size and alignment can vary.
- Do not use `unsafe.String`, `unsafe.Slice`, pointer arithmetic, or `uintptr` tricks to avoid allocation without measurement and review.
- Do not use unsafe to mutate immutable data or bypass visibility.
- Critical business/legal/security code should almost never contain unsafe.

## 18. cgo and FFI Gate

## Recommendation

cgo and FFI are high-risk integration code.

They must be isolated behind safe Go APIs.

## Always do

- Keep cgo bindings outside domain logic.
- Keep foreign calls in small adapter packages.
- Validate inputs and outputs across the boundary.
- Document ownership rules.
- Document lifetime rules.
- Document thread-safety assumptions.
- Test success and failure paths.
- Handle null pointers safely.
- Preserve error information from foreign code.
- Define who allocates and who frees memory.
- Avoid panics crossing FFI boundaries.
- Document CGO_ENABLED deployment assumptions.

## Prefer

- Pure Go alternatives when practical.
- Generated bindings only when deterministic and reviewed.
- Safe wrapper types.
- Explicit ownership transfer functions.
- C.CString/C.free patterns with clear cleanup.
- Integration tests with controlled fixtures.
- Clear conversion layer from FFI types to domain/application types.
- Platform-specific files with explicit build tags.

## Avoid

- Passing Go pointers across C boundaries unsafely.
- Assuming C strings are valid UTF-8.
- Ignoring foreign error codes.
- Mixing allocation/deallocation incorrectly.
- Letting foreign pointers escape into domain code.
- Exposing raw cgo in public application APIs.
- Unclear ownership of buffers.

## Almost never do

- Expose raw FFI directly to application/domain packages.
- Use cgo without safety documentation.
- Treat cgo code as normal low-risk Go.
- Let foreign code decide business outcomes without validation.
- Add cgo to a portable service without documenting build/deployment impact.


### Second-pass Go hardening

- cgo changes build reproducibility, cross-compilation, container images, deployment, memory ownership, and security posture.
- Keep cgo isolated behind Go interfaces or adapter packages.
- Document C library version, headers, linker flags, platform support, and deployment requirements.
- Validate all inputs and outputs crossing the C boundary.
- Define ownership of every allocated buffer and string.
- Never let C pointers escape into domain/application packages.
- Avoid calling Go callbacks from C unless the lifecycle and threading model are reviewed.
- Handle null pointers, error codes, errno, and partial failures explicitly.
- Avoid panics crossing cgo boundaries.
- Test cgo code on every supported target platform or document unsupported combinations.

## 19. Concurrency and Thread Safety

## Recommendation

Concurrency must be explicit, bounded, observable, and tested.

Go makes goroutines and channels easy to create. It does not prevent deadlocks, starvation, goroutine leaks, data races, unbounded queues, lost updates, cancellation bugs, or broken invariants.

## Always do

- Keep shared mutable state minimal.
- Prefer message passing or ownership transfer.
- Use synchronization primitives deliberately.
- Define lock ownership and lock ordering.
- Avoid holding locks across slow operations.
- Set timeouts where blocking/waiting occurs.
- Test concurrency-sensitive behavior.
- Run race detector where supported and practical.
- Make idempotency explicit for repeated/asynchronous processing.
- Keep critical sections small.
- Document concurrency assumptions.

## Prefer

- Immutable data shared safely.
- Channels for ownership transfer where they clarify flow.
- sync.Mutex/RWMutex only when shared mutation is necessary.
- sync/atomic only for simple, well-understood state.
- errgroup or project-approved structured goroutine management.
- Bounded queues.
- Context-aware workers.
- Dedicated state machines for concurrent workflows.
- Tests that prove cancellation and shutdown behavior.

## Avoid

- Global shared mutable state.
- Unbounded channels.
- Detached goroutines with no supervision.
- Blocking operations without cancellation.
- Locking multiple mutexes without ordering.
- Long critical sections.
- Copying values containing mutexes.
- Assuming “channel used” means concurrency is correct.
- Using sync.Map by default instead of designing ownership.

## Almost never do

- Fix races with sleeps.
- Hold a lock while performing network, database, signing, XML, or filesystem operations.
- Use unsafe concurrency primitives.
- Spawn goroutines that can fail silently.
- Ignore cancellation in critical workflows.
- Build legal/eSocial/SST workflows on unsupervised background goroutines.


### Second-pass Go hardening

- A program with no data races can still have deadlocks, goroutine leaks, lost updates, starvation, unbounded queues, and broken cancellation.
- Shared maps require synchronization when accessed by multiple goroutines and at least one goroutine writes.
- Define ownership for every channel: who sends, who closes, who receives, and when.
- Do not close channels from the receiver side unless that receiver owns all sends.
- Avoid unbounded goroutine creation from request handlers, queue consumers, or batch loops.
- Avoid holding locks while performing network, database, filesystem, signing, XML, or slow operations.
- Avoid copying lock-containing structs.
- Use `sync.Mutex`, `sync.RWMutex`, channels, atomics, or ownership transfer based on the actual concurrency model, not fashion.
- Race detector evidence is required for concurrency-sensitive changes where supported.
- Add deterministic tests for cancellation, shutdown, and repeated processing.

## 20. Goroutines, Context, and Cancellation

## Recommendation

Goroutines and context must be designed, not sprinkled.

Concurrent Go code has failure modes: cancellation leaks, blocking sends, task leaks, unbounded concurrency, hidden panics, non-idempotent retries, lost errors, and shutdown races.

## Always do

- Use goroutines only when they solve a concurrency or I/O problem.
- Pass context.Context through I/O and application boundaries where cancellation/deadlines matter.
- Do not store context.Context in structs except in rare, documented lifecycle types.
- Always call cancel functions returned by context.WithCancel/WithTimeout/WithDeadline when required.
- Use timeouts around external I/O.
- Handle cancellation deliberately.
- Bound concurrency.
- Propagate errors from goroutines.
- Avoid detached fire-and-forget goroutines.
- Test success, failure, timeout, and cancellation paths.
- Document retry and cancellation semantics.

## Prefer

- errgroup.WithContext for related goroutines when project-approved.
- Worker pools with explicit shutdown.
- Semaphores/channels for concurrency limits.
- select with explicit cancellation behavior.
- Retry policies with idempotency.
- Structured logging/tracing around goroutine lifecycles.
- Dedicated workers with explicit stop/drain semantics.
- Bounded channels.

## Avoid

- goroutine per item with no limit.
- Infinite retries.
- Unbounded channel sends.
- Sending on channels without cancellation path.
- Closing channels from receivers unless ownership is explicit.
- Runtime-specific or infrastructure concerns in domain logic.
- Swallowing goroutine errors.
- Ignoring panics in goroutines.

## Almost never do

- Hide failed goroutines.
- Use sleeps for synchronization.
- Retry non-idempotent operations without protection.
- Make legal/eSocial/SST sending fire-and-forget.
- Ignore timeout/cancellation behavior in external integrations.
- Leak goroutines after request completion.


### Second-pass Go hardening

- Every goroutine must have a bounded lifetime or a documented service lifetime.
- Every goroutine that can fail must have a supervised error path.
- Use `context.Context` as the first parameter for operations that can block, perform I/O, wait, or be canceled.
- Do not store context in long-lived structs unless the type itself represents a request/lifecycle and the project accepts it.
- Do not pass nil context; use `context.Background()` or `context.TODO()` only at composition boundaries, and prefer real request contexts.
- Always call cancel functions returned by `context.WithCancel`, `context.WithTimeout`, or `context.WithDeadline` when appropriate.
- Do not ignore `ctx.Err()` in loops or retries.
- Ensure retries stop on cancellation and respect deadlines.
- Use `time.NewTimer`/`time.NewTicker` carefully and stop/drain them where needed.
- Prefer structured concurrency patterns such as `errgroup` when the project allows the dependency.

## Cancellation evidence

- Test cancellation before work starts.
- Test cancellation during external wait.
- Test deadline expiration.
- Test shutdown drains or stops workers.
- Test that failed child work propagates to the caller.

## 21. Interfaces and Generics

## Recommendation

Interfaces and generics should model real variation or boundaries.

Do not create abstractions only because generated code looks more sophisticated.

## Always do

- Keep interfaces small.
- Accept interfaces and return concrete types where idiomatic.
- Put interfaces near the consumer when they represent ports.
- Use concrete types when there is no meaningful abstraction.
- Use generics when they reduce duplication without reducing clarity.
- Document public interfaces.
- Keep type constraints readable.
- Avoid exposing unnecessary type parameters.
- Prefer domain-specific interfaces over vague generic interfaces.

## Prefer

- Interfaces for external boundaries:
  - repository
  - clock
  - signer
  - event sender
  - storage
  - message publisher
- Concrete domain types.
- Generics for collections/algorithms with clear benefit.
- Type sets only when they express real constraints.
- Compile-time interface assertions for adapter conformance when useful.
- Unexported interfaces when only internal packages should implement them.

## Avoid

- Creating interfaces for every struct.
- Single-implementation interfaces without boundary/testing value.
- Overly generic domain code.
- Complex constraints in public APIs.
- interface{} / any when concrete types are known.
- Generics when an enum-like type would model closed variants better.
- Mock-driven interfaces that do not reflect real design.

## Almost never do

- Use interfaces to hide poor architecture.
- Use generics to avoid modeling business concepts.
- Expose complex generic APIs in application code without a real need.
- Make domain logic depend on dynamic dispatch by accident.
- Create package cycles and then add interfaces merely to break them superficially.


### Second-pass Go hardening

- Accept interfaces and return concrete types when that keeps ownership and behavior clear.
- Define interfaces near consumers, not providers, unless the interface is truly public API.
- Keep interfaces small and behavior-focused.
- Do not create an interface for every struct.
- Do not use interfaces only to make tests mock everything.
- Prefer concrete domain types inside domain logic.
- Use generics only when they reduce duplication without hiding business meaning.
- Avoid `interface{}`/`any` in domain APIs when a real type can be modeled.
- Avoid generic constraints that are so broad they provide no safety.
- Test public generic APIs with representative type arguments.

## 22. Code Generation, Reflection, and Templates

## Recommendation

Go has no macros. Code generation, reflection, and templates must be rare, deterministic, and justified.

Use generation to remove genuine repetition only when functions, interfaces, builders, or hand-written code are worse.

## Always do

- Prefer ordinary Go before code generation or reflection.
- Keep generated code deterministic.
- Mark generated files with the standard generated-code comment.
- Commit generated code if project policy requires it.
- Test generated behavior.
- Document generation commands.
- Avoid hiding business rules in generated code.
- Keep reflection usage small and tested.
- Keep generated code reviewable.

## Prefer

- go generate only for explicit project-approved generation.
- Deterministic inputs and outputs.
- Golden tests for generated payloads.
- Compile tests for generated code.
- Templates for boring transport/schema code, not business rules.
- Schema-specific generation when external contracts require it.

## Avoid

- Reflection for normal control flow.
- Reflection that hides side effects.
- Code generation that produces large unreviewable code.
- Generating domain logic from accidental transport shapes.
- Generator outputs that depend on local paths or timestamps.
- Generated unsafe/cgo code without explicit review.

## Almost never do

- Hide business validation inside generated code.
- Use reflection to bypass type modeling.
- Generate unsafe code without explicit review and tests.
- Add a generator casually.
- Hide legal/regulatory behavior in template expansion.


### Second-pass Go hardening

- Reflection, templates, and code generation must not hide business rules.
- Reflection-heavy code needs tests for missing fields, wrong types, nils, zero values, and unsupported inputs.
- Generated code must be deterministic and reviewable.
- `go generate` commands must be documented in source comments and PHASE-RESULT.md when used.
- Do not generate code that depends on current timestamp, absolute local paths, network state, or nondeterministic map order.
- Avoid template-based SQL/XML/JSON generation for legal/business payloads unless output is contract-tested.
- Keep generated code clearly marked and keep hand-written changes out of generated files.
- Review generated public APIs before accepting them as stable.
- Ensure generated mocks do not replace behavior tests with mock-interaction tests only.
- Add golden tests or diff checks for generated artifacts when output stability matters.

## 23. Serialization and Deserialization

## Recommendation

Serialization is a boundary concern.

Do not let encoding/json, XML packages, struct tags, or external payload shapes define the domain model accidentally.

## Always do

- Use DTOs at boundaries.
- Validate decoded data before domain use.
- Keep domain invariants independent from serialization.
- Test payload shape when it is part of the contract.
- Treat unknown fields deliberately.
- Treat missing fields deliberately.
- Keep versioning explicit.
- Avoid exposing internal types unintentionally.
- Make default values explicit and tested.
- Test serialization and deserialization errors.

## Prefer

- Dedicated request/response structs.
- Dedicated XML/event structs for legal payloads.
- Explicit mappers from DTOs to domain types.
- Golden tests for stable payloads.
- Schema validation where applicable.
- json.Decoder.DisallowUnknownFields when strict input is required.
- Backward-compatible decoding where required by API versioning.
- Stable date/time formats.
- Explicit enum string validation.

## Avoid

- Using domain structs as API DTOs by default.
- map[string]any as business data.
- Silent zero values for required business fields.
- Hiding validation in struct tags only.
- Accidental field renames.
- Accidental date/time format changes.
- Untested custom MarshalJSON/UnmarshalJSON/XML methods.
- Leaking internal enum values into public contracts.

## Almost never do

- Decode untrusted payloads directly into domain objects.
- Treat decoding success as business validation.
- Use generic maps for legal/eSocial/SST payloads.
- Allow invalid legal events to exist because JSON/XML accepted the shape.
- Encode legal/business compatibility through undocumented tag behavior.


### Second-pass Go hardening

- Serialization belongs at boundaries; domain invariants must not depend on `encoding/json`, `encoding/xml`, protobuf, or database tags alone.
- Use DTOs for external contracts and map to validated domain types.
- For strict JSON input, use decoder settings that reject unknown fields where the API requires strictness.
- Treat `omitempty` carefully because it can hide false, zero, empty, or nil values that may be meaningful.
- Test required, optional, unknown, null, empty, malformed, and versioned payloads.
- Test time/date formats and timezone behavior.
- Avoid exposing internal domain fields accidentally through exported struct fields and tags.
- Do not deserialize untrusted payloads into `map[string]any` and then make business decisions from ad hoc assertions.
- Preserve backward compatibility intentionally for public APIs.
- Add golden tests for stable legal/API payloads.

## 24. XML Safety

## Recommendation

XML processing must be hardened.

This is especially important for eSocial/SST and any legal/regulatory payload.

## Always do

- Use a dedicated XML library/encoder/parser.
- Avoid XML string concatenation.
- Validate XML against the expected schema where applicable.
- Keep XML generation deterministic.
- Test namespaces.
- Test encoding.
- Test required and optional fields.
- Test malformed XML.
- Test external rejection paths.
- Redact sensitive XML in logs.
- Keep layout/schema version explicit.
- Keep canonicalization requirements explicit when signatures are involved.

## Prefer

- Version-specific XML models.
- Golden tests for generated XML.
- Schema validation tests.
- Canonicalization tests when signatures are involved.
- Contract tests for external XML payloads.
- Dedicated fixtures per event type.
- Safe parser settings.
- Streaming parsing for large payloads.
- XML comparison helpers that understand namespaces/canonicalization.

## Avoid

- Building XML with fmt.Sprintf and string concatenation.
- Mixing schema versions in one unstructured builder.
- Ignoring namespaces.
- Ignoring canonicalization when signatures matter.
- Comparing XML as raw strings when canonical comparison is required.
- Logging complete sensitive XML.
- Using transport XML structs as domain models.
- Letting XML builder code decide business validity.

## Almost never do

- Generate legal XML without golden tests.
- Sign XML without deterministic canonicalization evidence.
- Send XML without validation when schema is available.
- Put legal/business rules inside XML builder code.
- Use unversioned XML builders for versioned legal layouts.
- Treat XML generation as a string formatting problem.


### Second-pass Go hardening

- XML is a legal/contract boundary, not a string-formatting exercise.
- Use explicit structs/builders and deterministic output for legal XML.
- Validate namespaces, element ordering, attributes, encoding, and schema versions.
- Do not build XML with string concatenation.
- Treat XML input as untrusted; test malformed, oversized, unexpected namespace, and missing-field cases.
- Do not log raw XML containing personal, health, payroll, certificate, or legal data without redaction policy.
- Keep canonicalization and signature requirements explicit when XML signatures are involved.
- Compare XML with namespace-aware/canonical helpers when raw string equality is misleading.
- Keep XML DTOs separate from domain rules.
- Add golden fixtures for every critical generated XML variant.

## 25. Date, Time, Time Zones, and Clock

## Recommendation

Date/time bugs are business bugs.

Use explicit types and freeze time in tests.

## Always do

- Use time.Time deliberately.
- Use date-only value objects for date-only business concepts.
- Inject a clock/time provider when current time affects behavior.
- Test time-dependent logic with fixed time.
- Define timezone policy.
- Define inclusive/exclusive date range semantics.
- Validate date ranges.
- Test boundary dates.
- Avoid relying on local machine timezone.
- Keep monotonic-clock behavior in mind when comparing serialized/deserialized times.

## Prefer

- Domain value objects for:
  - legal dates
  - event periods
  - deadlines
  - validity windows
  - transmission timestamps
- time.Time at technical boundaries only when timestamp semantics are required.
- ISO-8601/RFC3339 at technical boundaries unless integration requires another format.
- Explicit timezone conversion at boundaries.
- Tests for:
  - same-day boundaries
  - end-of-month
  - leap year
  - invalid ranges
  - timezone conversion
  - daylight saving behavior when relevant

## Avoid

- Hidden local timezone assumptions.
- Comparing dates as strings.
- Parsing dates repeatedly inside business rules.
- Mixing date-only and timestamp concepts.
- Using current real time in deterministic tests.
- Silent fallback on invalid dates.
- Storing external date strings directly in domain objects.

## Almost never do

- Use local machine time as business truth.
- Ignore timezone requirements in legal/eSocial/SST events.
- Let external payload date strings leak into domain logic.
- Make tests depend on today’s date.
- Use time.Now directly in business rules.


### Second-pass Go hardening

- Use `time.Time` for instants and date/time values, but define domain types for date-only legal concepts when needed.
- Do not rely on the machine local timezone for business rules.
- Inject a clock when current time affects behavior.
- Use fixed clocks in tests.
- Define inclusive/exclusive boundaries for periods and deadlines.
- Use `Time.Equal` rather than `==` when location or monotonic clock fields may differ.
- Strip or normalize monotonic components when serializing/comparing persisted timestamps.
- Validate zero `time.Time` if zero is not a valid business value.
- Test leap years, month boundaries, deadline boundaries, timezone conversions, and daylight-saving behavior when relevant.
- Keep external date strings out of domain logic; parse at boundaries.

## 26. Money, Decimals, Measurements, and Numeric Rules

## Recommendation

Use exact and domain-appropriate numeric types.

Go primitive numeric types are not enough for all business/legal contexts.

## Always do

- Define numeric units explicitly.
- Avoid magic numbers.
- Test boundary values.
- Test rounding rules.
- Test minimum/maximum values.
- Avoid floating point for money.
- Use checked/safe arithmetic where overflow matters.
- Validate measurements and legal thresholds.
- Document rounding policies.
- Test zero, negative, maximum, and fractional cases where applicable.

## Prefer

- Domain value objects for:
  - money
  - percentages
  - measurements
  - rates
  - quantities
  - thresholds
  - exposure levels
- Decimal packages for money/precise decimals when project-approved.
- Integer minor units for money when compatible with business rules.
- Explicit rounding policies.
- Constants named after business meaning.
- Types/names that include units:
  - durationDays
  - amountCents
  - noiseExposureDB
  - ratePercent
- Overflow checks where audit/legal/security calculations need them.

## Avoid

- float32/float64 for money.
- Hidden unit conversion.
- Silent overflow.
- Numeric literals spread across code.
- Rounding inside unrelated functions.
- Comparing floats directly.
- Unclear measurement units.
- Using int for all domain quantities without range consideration.

## Almost never do

- Round legal/payroll/financial values without tests.
- Mix units in the same field.
- Use binary floating-point for legal, payroll, or financial calculations.
- Treat measurement units as comments instead of types/names.
- Ignore overflow risk in audit/legal calculations.


### Second-pass Go hardening

- Use exact numeric representation for money, payroll, legal thresholds, and measurements.
- Avoid `float32`/`float64` for money or legal/payroll calculations.
- Prefer integer minor units or approved decimal libraries when exact decimals are required.
- Name units in types, fields, constants, and tests.
- Validate ranges and units before constructing domain values.
- Define rounding policy at the domain level and test it.
- Check overflow where values may grow through multiplication, aggregation, conversion, or external input.
- Avoid silent unit conversions in mapper code.
- Test zero, negative, maximum, fractional, rounding, and invalid-unit cases.
- Keep magic numeric literals out of legal/business logic.

## 27. Collections and Iteration

## Recommendation

Use the clearest collection construct, not the cleverest.

Go loops are a strength. Use them when they make business rules obvious.

## Always do

- Choose collection types by behavior.
- Make ordering explicit.
- Use maps for repeated lookup.
- Avoid unnecessary allocation.
- Avoid cloning large collections without reason.
- Preserve deterministic output when tests/contracts depend on ordering.
- Protect maps from concurrent writes.
- Copy slices/maps at boundaries when ownership is unclear.

## Prefer

- Slices for ordered collections.
- Maps for keyed lookup.
- Struct sets or map[T]struct{} for membership.
- Sorting keys before deterministic output.
- Plain loops for complex branching.
- Preallocation when size is known and performance matters.
- Small helper functions for repeated transformations.

## Avoid

- Hidden side effects inside helper iteration functions.
- Accidental nondeterministic map iteration output.
- Repeated slice scans in performance-sensitive paths.
- Appending to caller-owned slices without documenting ownership.
- Mutating maps while iterating in confusing ways.
- Cloning because ownership was not designed.

## Almost never do

- Hide persistence, network, signing, or message publishing side effects inside collection helpers.
- Depend on map iteration order.
- Clone entire payloads just to simplify control flow.
- Turn business workflows into obscure functional-style helpers.


### Second-pass Go hardening

- Map iteration order is not deterministic; sort keys when output, tests, signatures, XML, JSON, audit records, or hashes require stable ordering.
- Appending to slices can alias caller-owned arrays; copy when ownership must not be shared.
- Nil slices and empty slices may serialize differently depending on contract.
- Nil maps panic on assignment.
- Avoid modifying a slice while ranging over it unless the behavior is carefully designed and tested.
- Avoid hidden side effects inside functional-style helpers.
- Use plain loops when business logic has branching or error handling.
- Preallocate slices/maps only when it improves clarity or measured performance.
- Protect shared collections with synchronization or ownership transfer.
- Test deterministic ordering for generated payloads and audit outputs.

## 28. Logging and Observability

## Recommendation

Logs are operational evidence, not decoration.

For concurrent services, external integrations, and legal/eSocial/SST operations, structured logging and tracing are strongly preferred.

## Always do

- Use the project-approved logging/tracing system.
- Use structured fields.
- Include correlation IDs where available.
- Never log secrets.
- Never log private keys.
- Never log passwords.
- Never log tokens.
- Never log raw sensitive legal/personnel/health payloads without explicit redaction.
- Log failures with useful context.
- Avoid logging the same error repeatedly.
- Make goroutine failures observable.
- Ensure logs do not become the only audit trail.

## Prefer

- slog, zap, zerolog, or project-approved structured logging.
- OpenTelemetry or project-approved tracing where distributed tracing is required.
- Stable event names.
- Domain identifiers instead of raw payloads.
- Redaction utilities.
- Separate audit trail from debug logs.
- Clear log levels:
  - ERROR for failed operations requiring attention
  - WARN for degraded or unexpected recoverable states
  - INFO for important business/operational events
  - DEBUG for development diagnostics
- Explicit fields for:
  - request ID
  - correlation ID
  - event ID
  - batch ID
  - tenant/account ID where safe
  - external protocol/receipt where safe

## Avoid

- fmt.Println in production services.
- log.Printf debug leftovers.
- Stringly logs with no structured fields.
- Logging whole XML/JSON payloads.
- Logging inside tight loops without rate control.
- Vague messages like "failed" without context.
- Logging sensitive values through %+v.
- Automatically logging full structs that may contain secrets.

## Almost never do

- Log sensitive eSocial/SST XML unredacted.
- Use logs as the only audit trail.
- Hide failures because they were logged.
- Let goroutines fail without traceability.
- Add String methods for secret-bearing types without redaction strategy.


### Second-pass Go hardening

- Prefer structured logging for services; `log/slog` is available in modern Go standard library versions, while project-approved libraries may also be used.
- Logs must contain useful context without leaking secrets or sensitive personal/legal/health data.
- Use request IDs, correlation IDs, tenant/account IDs, event IDs, batch IDs, and external receipt/protocol numbers where safe.
- Do not log private keys, passwords, tokens, certificate material, raw sensitive XML/JSON, or full personnel/health records.
- Avoid `fmt.Println`, `log.Println`, and debug dumps in production paths.
- Avoid logging the same error repeatedly at every layer.
- Make background goroutine failures observable.
- Logs are not a substitute for audit records.
- Redact via explicit types or logging helpers; do not rely on humans remembering not to log fields.
- Test redaction for critical secret-bearing or sensitive payload types.

## 29. Auditability and Traceability

## Recommendation

For important business operations, especially legal/regulatory operations, auditability is part of correctness.

## Always do

- Record who/what initiated important actions when available.
- Record when important actions occurred.
- Record what business object was affected.
- Record result status.
- Preserve correlation IDs.
- Preserve external identifiers such as:
  - receipts
  - batch IDs
  - protocol numbers
  - rejection codes
  - signatures
  - certificate identifiers where safe
- Make audit events queryable.
- Test audit behavior for critical flows.
- Record failure paths, not only success paths.
- Keep audit records stable enough for support/investigation.

## Prefer

- Explicit audit records.
- Append-only audit history for legal events.
- Stable audit event names.
- Redacted payload references.
- Hashes/checksums for generated documents when useful.
- Traceable state transitions.
- Separate audit data from debug logs.
- Typed audit event constants.
- Audit event builders that enforce required fields.
- Tests for audit completeness.

## Avoid

- Audit data only in logs.
- Free-text-only audit records.
- Missing failure audit records.
- Inconsistent event names.
- Storing more sensitive data than necessary.
- Losing the link between request, generated payload, sent payload, response, and final state.
- Making audit optional for critical flows.

## Almost never do

- Send, sign, cancel, correct, or interpret legal events without traceability.
- Make audit optional for eSocial/SST flows.
- Lose the connection between generated payload and transmission result.
- Store sensitive payloads without redaction/encryption policy.
- Treat audit trail as post-release polish.


### Second-pass Go hardening

- Auditability is part of correctness for legal, payroll, health/SST, security, signing, transmission, cancellation, and correction flows.
- Audit events should be explicit records, not only logs.
- Record actor, action, object, timestamp, result, correlation ID, and external identifiers where available and safe.
- Record failure paths as well as success paths.
- Preserve links between generated payload, signed payload, sent payload, provider response, receipt, rejection, and final state.
- Keep audit events stable enough for support and investigation.
- Avoid free-text-only audit records for critical workflows.
- Avoid storing more sensitive payload data than necessary.
- Test that audit records are written for critical success and failure paths.
- Document any intentionally unaudited critical action as a quality risk.

## 30. Security Baseline

## Recommendation

Security must be default behavior, not cleanup work.

The implementation LLM must assume inputs are untrusted unless proven otherwise.

## Always do

- Validate external input.
- Avoid injection vulnerabilities.
- Use parameterized database queries.
- Protect secrets.
- Use TLS verification.
- Set network timeouts.
- Avoid unsafe deserialization patterns.
- Use maintained cryptographic libraries.
- Run vulnerability/security checks when configured.
- Keep permissions narrow.
- Redact sensitive logs.
- Avoid unnecessary dependencies.
- Treat file paths, XML, JSON, CLI args, environment variables, and network responses as untrusted inputs.

## Prefer

- govulncheck.
- gosec when project-approved.
- Minimal dependency graph.
- crypto/tls and project-approved TLS configuration.
- Explicit secret loading from environment/secret manager.
- Centralized crypto/signing adapters.
- Strong typed inputs before business use.
- Fuzz/property tests for parsers.
- Allow-lists for constrained values.
- Explicit limits on payload sizes.

## Avoid

- Hardcoded secrets.
- Disabling TLS verification.
- Building SQL with string concatenation.
- Parsing untrusted data into overly permissive structures.
- Logging secrets.
- Adding abandoned modules.
- Using unmaintained crypto modules.
- Ignoring security advisories.
- Accepting arbitrary file paths without validation.
- Shelling out with untrusted arguments.
- Unbounded decompression/parsing.

## Almost never do

- Implement custom cryptography.
- Store private keys/certificates in source control.
- Use weak hashing for security-sensitive purposes.
- Suppress critical/high vulnerability findings without documented justification.
- Disable security checks to finish the phase.
- Treat development environments as permission to write insecure code.
- Use unsafe/cgo to parse untrusted input without fuzzing/safety review.


### Second-pass Go hardening

- Treat all external input as untrusted: HTTP, CLI, files, environment, XML, JSON, database rows, queue messages, and provider responses.
- Use parameterized queries and safe encoders.
- Do not disable TLS verification.
- Do not shell out with untrusted arguments unless arguments are strictly validated and invocation is necessary.
- Set payload size limits, timeouts, and retry limits for network/input processing.
- Protect secrets in memory, logs, errors, test fixtures, and generated files.
- Use maintained cryptographic libraries; do not invent crypto.
- Run `govulncheck` or configured vulnerability tooling after meaningful dependency/security changes.
- Avoid broad filesystem access and path traversal risks.
- Add negative tests for injection, invalid input, malformed payloads, and authorization boundaries where relevant.

## 31. Dependency and Supply Chain Hygiene

## Recommendation

Every Go module dependency is a liability until justified.

Dependencies affect security, compile time, binary size, Go version requirements, licensing, transitive risk, auditability, and maintenance burden.

## Always do

- Add dependencies only with a clear purpose.
- Prefer mature, maintained modules.
- Keep module graph minimal.
- Check why dependencies are needed when dependency changes are significant.
- Run supply-chain checks when configured.
- Document major dependency additions in PHASE-RESULT.md.
- Review license compatibility when adding dependencies.
- Review whether the dependency brings cgo, unsafe, native code, or generation steps.
- Review whether the dependency affects the required Go version.

## Prefer

- Standard library when it is sufficient and clear.
- govulncheck for vulnerability reachability checks.
- go list -m all for module graph review.
- go mod why for dependency justification.
- go-licenses or project-approved license tooling.
- Minimal features/configuration in dependency packages.
- Dependency wrappers/adapters for external systems.
- tools.go or pinned tool install policy for development tools.

## Avoid

- Adding a module for trivial code.
- Pulling HTTP/database/XML dependencies into domain packages.
- Abandoned modules.
- Multiple modules solving the same problem.
- Native/cgo dependencies when a safe pure-Go option is sufficient.
- Unknown generators without reason.
- Dependencies that change public API shape accidentally.

## Almost never do

- Ignore known security advisories.
- Add dependencies with incompatible licenses.
- Add a dependency that requires a higher Go version without documentation.
- Modify supply-chain configuration only to hide a problem.
- Depend on unmaintained crypto, XML, parsing, or TLS modules for critical code.
- Use a fork or replace directive indefinitely without ownership and update policy.


### Second-pass Go hardening

- Every dependency must have a purpose, owner, and risk profile.
- Prefer the standard library when it is sufficient and clear.
- Review maintenance activity, license, transitive dependency graph, known vulnerabilities, and cgo/unsafe usage before adding a dependency.
- Do not add large frameworks to avoid writing small domain code.
- Avoid dependencies that require unexpected global state, background goroutines, network access during init, or hidden code generation.
- Use `go mod why -m` to explain non-obvious modules.
- Review major version import paths, especially `/v2+` modules.
- Avoid `replace` forks unless the risk and exit plan are explicit.
- Keep dependency updates targeted unless the phase is dependency maintenance.
- Document dependency additions and removals in PHASE-RESULT.md.

## 32. Build Tags and Conditional Compilation

## Recommendation

Go build tags are part of the public build contract.

Poor build-tag design creates fragile builds and hidden behavior differences.

## Always do

- Keep build tags explicit and documented.
- Test important tag combinations.
- Keep default build behavior correct.
- Avoid hidden business behavior changes through tags.
- Ensure normal go test ./... works unless impossible and documented.
- Ensure platform-specific files compile for intended targets.
- Keep tag names stable for public/reusable packages.
- Document cgo-dependent tags.

## Prefer

- Build tags based on capability or platform.
- Small, orthogonal tags.
- CI checks for:
  - default build
  - cgo-enabled/disabled when relevant
  - platform targets
  - integration tags
  - important combinations
- Runtime configuration over build tags for business behavior.
- Versioned modules/packages for legal schemas instead of build tags.

## Avoid

- Mutually exclusive tags unless unavoidable.
- Tags that silently change legal/business rules.
- Tags that change public API unexpectedly.
- Large default tag assumptions.
- Tag names based only on implementation details.
- Tag combinations that compile but are not tested.
- Tag combinations that are impossible but undocumented.

## Almost never do

- Use build tags to hide broken code.
- Make eSocial/SST layout behavior ambiguous through build tags.
- Use build tags instead of explicit versioned modules for legal schemas.
- Break semver through tag behavior without documenting impact.
- Depend on developer-local tags to make normal builds work.


### Second-pass Go hardening

- Build tags are part of the build contract; every tag path affected by a phase must be tested.
- Use modern `//go:build` lines and keep legacy `// +build` only when project policy requires it.
- Keep tag names capability-based and documented.
- Avoid using build tags to hide broken code from default tests.
- Avoid legal/business rule differences behind undocumented tags.
- Test `CGO_ENABLED=0` and `CGO_ENABLED=1` when both are expected to work.
- Test platform-specific files for target `GOOS`/`GOARCH` values.
- Keep test-only tags from leaking into production behavior.
- Document mutually exclusive tags and impossible combinations.
- Ensure generated code respects build-tag constraints.

## 33. Public API Design

## Recommendation

Public Go APIs should be clear, type-safe, documented, and semver-aware.

Use idiomatic Go and stable module versioning as the baseline for public modules and reusable internal modules.

## Always do

- Keep public API minimal.
- Use meaningful types.
- Avoid exposing implementation details.
- Document exported types/functions in reusable packages.
- Preserve semver compatibility for published/reused modules.
- Test public API behavior.
- Avoid exported fields unless intentionally stable.
- Keep error semantics clear.
- Avoid exposing framework-specific types unless the package is specifically a framework adapter.
- Respect module major versioning for breaking changes.

## Prefer

- Defined types for meaningful distinctions.
- Constructors for invariant enforcement.
- Options pattern only when it improves clarity.
- Error values/types with meaningful comparison behavior.
- Public interfaces only when external implementation is intended.
- Internal packages for implementation details.
- Examples as documentation tests where useful.
- Clear package comments for public packages.

## Avoid

- Public API exposing internal packages indirectly.
- Public structs with many mutable fields.
- Public APIs returning vague strings.
- Boolean parameters in public APIs.
- Breaking changes without documentation.
- Exposing database/client/framework types in domain APIs.
- Re-exporting dependencies as part of public API accidentally.
- Making generated code the public API without review.

## Almost never do

- Publish or reuse a module with undocumented critical public APIs.
- Break semver silently.
- Expose legal/eSocial/SST internals as arbitrary maps.
- Use public API shape generated by accident.
- Change public error behavior without documenting compatibility impact.


### Second-pass Go hardening

- Public API includes exported identifiers, package paths, error semantics, JSON/XML contracts, CLI flags, configuration keys, and database migration expectations.
- Export only what must be used outside the package.
- Document exported identifiers in reusable/public packages.
- Preserve backward compatibility or document breaking changes.
- Follow Go module semantic import versioning for major version changes.
- Avoid exposing framework, database, or provider-specific types from domain APIs.
- Avoid exported mutable fields when invariants matter.
- Avoid returning vague errors that callers cannot inspect.
- Use examples/tests to lock public behavior where practical.
- Do not let generated APIs become public without review.

## 34. Documentation and Examples

## Recommendation

Documentation must help future maintainers use and verify the code.

For public APIs, examples should compile when practical.

## Always do

- Update documentation when behavior changes.
- Document exported APIs in reusable packages.
- Document important invariants.
- Document error semantics.
- Document unsafe/cgo requirements.
- Keep command examples accurate.
- Document skipped quality gates.
- Document deviations from architecture.
- Keep docs close to the code they explain.

## Prefer

- Package comments for important packages.
- Example tests for public examples.
- Error documentation for fallible functions.
- Panic documentation for functions that can panic.
- Safety documentation for unsafe/cgo boundaries.
- ADRs for architectural decisions.
- Golden fixtures as contract documentation.
- Short examples that show correct use.

## Avoid

- Documentation that restates obvious code.
- Outdated examples.
- Vague claims such as “secure” or “robust” without evidence.
- Large docs not connected to tests.
- Comments that excuse poor design.
- Public APIs with hidden invariants.
- Examples that require real external services.

## Almost never do

- Leave unsafe/cgo code undocumented.
- Leave legal/eSocial/SST assumptions undocumented.
- Change architecture without documenting why.
- Depend on comments instead of tests for business rules.
- Claim behavior in docs that tests do not verify.


### Second-pass Go hardening

- Documentation must match tested behavior.
- Exported packages and identifiers in reusable libraries need useful comments.
- Important functions should document errors, panics, concurrency safety, ownership, and context behavior where relevant.
- Use `ExampleXxx` tests when examples should compile and stay correct.
- Keep README command examples synchronized with actual commands.
- Document configuration keys, environment variables, build tags, and operational assumptions.
- Document unsafe/cgo safety requirements in code.
- Document legal/eSocial/SST assumptions close to the code and fixtures that enforce them.
- Avoid comments that merely restate code while omitting invariants.
- Do not claim code is secure, robust, or compliant without evidence.

## 35. Testing Strategy

## Recommendation

Tests must verify behavior, not implementation.

A generated test suite that only executes code without meaningful assertions is not quality evidence.

## Always do

- Add automated tests for every implemented business rule.
- Test success paths.
- Test failure paths.
- Test edge cases.
- Test boundary values.
- Test regression cases.
- Keep tests deterministic.
- Keep tests independent.
- Keep tests runnable from command line.
- Use clear test names.
- Avoid real external services in tests.
- Avoid real credentials in tests.
- Keep test fixtures understandable.

## Prefer

- Unit tests for domain logic.
- Integration tests for adapters.
- Contract tests for external APIs.
- Golden tests for generated payloads.
- Fuzz tests for parsers and rule-heavy validation.
- Fixed clock/time provider for time-dependent tests.
- Fixture builders for readability.
- httptest for HTTP clients/servers.
- testcontainers or equivalent for real infrastructure tests when useful.
- t.Run table tests when rule matrices are clear.
- t.Helper for helper functions.
- t.Cleanup for resource cleanup.

## Avoid

- Testing only happy paths.
- Assertion-free tests.
- Tests that duplicate production logic.
- Tests depending on execution order.
- Tests depending on current date/time.
- Tests depending on real credentials.
- Live external systems in automated tests.
- Flaky timing-based tests.
- Excessive snapshots that approve bad output.
- Mocking everything until no real behavior is tested.

## Almost never do

- Delete tests to make the phase pass.
- Lower thresholds to make the phase pass.
- Use sleeps in tests as synchronization.
- Leave flaky tests unresolved.
- Close a legal/eSocial/SST bug without a regression test.
- Claim quality because “Go test passed once.”


### Second-pass Go hardening

- Tests must verify behavior with assertions; compilation-only tests are not enough for implemented logic.
- Use table tests for rule matrices, but keep each case named and meaningful.
- Use `t.Helper()` in test helpers.
- Use `t.TempDir()` for filesystem tests.
- Use `t.Cleanup()` for cleanup that must run even after failure.
- Use fake clocks, fake servers, and fixtures instead of real time and real external services.
- Be careful with `t.Parallel()`; it must not race shared state, environment variables, ports, or global configuration.
- Avoid sleeps as synchronization; use channels, contexts, eventual assertions with deadlines, or fake clocks.
- Add tests for nil, zero, empty, malformed, boundary, and invalid inputs.
- Regression tests must fail without the bug fix.

## 36. Test Types Required by Risk

## Low-risk code

Examples:

- simple DTOs
- straightforward mappers
- simple configuration
- non-critical helper functions

Required evidence:

- build
- formatting
- go vet/static checks
- basic tests when behavior exists

## Medium-risk code

Examples:

- application services
- validation logic
- persistence adapters
- API endpoints
- non-critical integrations

Required evidence:

- unit tests
- integration tests where applicable
- coverage
- vet/static checks
- error-path tests
- boundary tests

## High-risk code

Examples:

- business rules
- payroll/legal logic
- health/SST logic
- event state transitions
- authorization
- audit-critical persistence
- goroutine workflows with retries
- important parsers

Required evidence:

- unit tests
- edge-case tests
- regression tests
- coverage thresholds
- mutation testing where available
- fuzz tests where useful
- architecture/boundary checks where applicable
- failure/retry/cancellation tests where applicable
- race detector where concurrency is involved

## Critical-risk code

Examples:

- eSocial/SST XML generation
- legal event validation
- digital signing
- event transmission
- receipt interpretation
- cancellation/correction flows
- certificate handling
- unsafe code
- cgo/FFI code
- security-sensitive code
- parsers for untrusted input

Required evidence:

- unit tests
- contract tests
- golden tests
- schema validation tests where applicable
- mutation testing or mutation-ready structure
- fuzz tests where applicable
- error-path tests
- audit tests
- redaction tests
- dependency/security checks
- race detector for concurrent paths
- explicit panic/unsafe/cgo review


### Second-pass Go hardening

- Risk level determines evidence depth, not code volume.
- Low-risk code may need only build, formatting, vet, and basic behavior tests.
- Medium-risk code needs error-path and boundary tests.
- High-risk code needs edge cases, regression tests, and coverage evidence.
- Critical-risk code needs contract/golden/error/audit/security evidence and often fuzz or mutation readiness.
- Concurrency-sensitive code requires race/cancellation/shutdown evidence.
- Public API changes require compatibility and documentation evidence.
- Serialization contract changes require golden or contract fixtures.
- Persistence changes require migration/query/mapping tests.
- Dependency/security changes require vulnerability/dependency evidence.

## 37. Coverage and Mutation Testing

## Recommendation

Coverage is necessary but not sufficient.

Mutation testing is stronger evidence for critical business rules because it checks whether tests detect behavioral changes.

## Default thresholds

| Area | Line Coverage | Branch/Behavior Coverage | Mutation Score |
|---|---:|---:|---:|
| Domain/business rules | >= 90% | >= 85% | >= 80% |
| Critical eSocial/SST/legal rules | >= 95% | >= 90% | >= 85% |
| Application services | >= 85% | >= 80% | >= 75% |
| Infrastructure adapters | >= 70% | >= 60% | When practical |
| API/handlers | >= 70% | >= 60% | Usually not required |
| Generated code | Exclude only with reason | Exclude only with reason | Exclude only with reason |

## Always do

- Measure coverage when tooling is available.
- Mutation-test critical rules when tooling is available.
- Document coverage results.
- Document mutation results.
- Document uncovered critical paths.
- Add tests before increasing quality score.
- Exclude generated code only with explicit reason.
- Avoid writing tests only to increase coverage numbers.

## Prefer

- go test -coverprofile for coverage.
- go tool cover -func for summary evidence.
- -covermode=atomic when concurrency is involved.
- Mutation testing focused on domain/application packages first.
- Smaller mutation scopes per phase.
- Regression tests that kill previously surviving mutants.
- Coverage reports attached or summarized in PHASE-RESULT.md.

## Avoid

- Treating high line coverage as proof of correctness.
- Assertion-free tests.
- Excluding difficult code without reason.
- Measuring only handler/framework code.
- Ignoring surviving mutants in critical rules.
- Counting generated code as meaningful coverage.
- Chasing 100% coverage in low-risk glue while ignoring critical logic.

## Almost never do

- Accept untested business rules.
- Accept untested eSocial/SST payload generation.
- Claim production-grade quality without coverage evidence.
- Claim production-grade quality for critical rules without mutation evidence or documented mutation-readiness.
- Score 90+ with critical surviving mutants unexplained.


### Second-pass Go hardening

- Coverage numbers are evidence, not proof.
- Use coverage to find untested branches in domain/application code first.
- Do not improve coverage by testing trivial getters while leaving business rules untested.
- Use `-coverpkg=./...` when cross-package use-case tests should count toward package coverage.
- Use atomic coverage mode when race/concurrency testing and coverage intersect.
- Mutation testing is especially valuable for legal/business rules, validation, state machines, and numeric rules.
- Surviving mutants in critical rules must be fixed with better tests or documented with a reason.
- Exclude generated code only with an explicit reason.
- Document uncovered critical branches.
- Do not score critical code highly without meaningful coverage and mutation/fuzz readiness evidence.

## 38. Fuzzing and Property-Style Tests

## Recommendation

Use Go fuzzing and property-style tests where example-based tests are insufficient.

They are especially useful for parsers, validators, state machines, legal rules, encoding/decoding, numeric rules, and unsafe/cgo boundaries.

## Always do

- Use deterministic seeds/corpus files where possible.
- Minimize failing cases.
- Save regression cases discovered by fuzz tests.
- Keep fuzz inputs constrained to meaningful input where practical.
- Test invariants, not implementation details.
- Keep fuzz targets focused.
- Document fuzz evidence for critical code.

## Prefer

- Native Go fuzz tests for parsers, binary formats, XML/JSON boundaries, and unsafe code.
- Property-style table/generator tests for:
  - validation invariants
  - round-trip serialization
  - date ranges
  - state machines
  - numeric boundaries
  - event lifecycle transitions
  - canonicalization rules
  - parse/serialize consistency
- Corpus files for important discovered cases.
- Short fuzz windows in normal CI and longer fuzzing in scheduled jobs.

## Avoid

- Random tests without reproducibility.
- Overly broad generators producing mostly meaningless input.
- Property tests with weak assertions.
- Treating fuzzing as a replacement for unit/contract/golden tests.
- Fuzz targets that require live services.
- Fuzzing huge workflows instead of focused parsers/validators.

## Almost never do

- Ignore a fuzz/property failure.
- Fail to convert discovered bugs into regression tests.
- Use fuzzing on sensitive production data.
- Claim parser/unsafe robustness without adversarial tests when risk is high.
- Use random sleeps or nondeterministic timing as “property testing.”


### Second-pass Go hardening

- Native Go fuzz tests are useful for parsers, validators, decoders, encoders, state machines, and untrusted input boundaries.
- Keep fuzz targets small and deterministic.
- Seed fuzz tests with real fixtures and important edge cases.
- Save discovered failures as regression corpus entries or normal tests.
- Avoid fuzz targets that require real databases, network services, credentials, or wall-clock assumptions.
- Use property-style assertions: round trips, invariants, idempotency, normalization, rejection rules, and no panics.
- Bound input sizes when the target could consume unbounded memory or CPU.
- Document fuzz time and target names in PHASE-RESULT.md for critical code.
- Fuzzing does not replace unit, contract, golden, and audit tests.
- Never ignore a fuzz failure because it is rare.

## 39. Static and Dynamic Analysis

## Recommendation

Go quality gates should combine compiler checks, gofmt, go vet, tests, race detector, coverage, fuzzing, mutation testing, and supply-chain analysis.

## Always do

- Run configured static/dynamic analysis.
- Fix high-confidence findings.
- Keep suppressions narrow.
- Document suppressions.
- Avoid broad exclusions.
- Treat new warnings as phase failures unless justified.
- Document unavailable tools.
- Prioritize findings in unsafe, cgo, legal, security, and audit code.

## Prefer

- gofmt.
- go vet.
- go test.
- go test -race.
- go test -coverprofile.
- go test -fuzz for focused fuzz targets.
- staticcheck.
- golangci-lint.
- govulncheck.
- gosec when configured.
- go list/go mod why for dependency reasoning.
- go doc/example tests for public API documentation checks.

## Avoid

- Suppressing warnings to avoid refactoring.
- Ignoring unchecked errors.
- Allowing new warnings because old warnings already exist.
- Running tools but ignoring their results.
- Treating race detector as optional when concurrency was introduced and it is supported.
- Running only go test and claiming full quality.

## Almost never do

- Disable go vet globally.
- Add broad nolint directives.
- Ignore dynamic analysis findings in unsafe/cgo code.
- Modify tool configuration only to hide generated problems.
- Score 90+ without static analysis evidence for non-trivial code.


### Second-pass Go hardening

- Combine compiler checks, tests, vet, staticcheck/lint, race detection, coverage, fuzzing, mutation testing, vulnerability checks, and profiling as risk requires.
- Prioritize high-confidence findings in domain, security, persistence, signing, XML, concurrency, and audit code.
- Keep suppressions close to the finding and explain why the code is safe.
- Do not modify linter configuration to hide new problems without documenting the quality impact.
- Race detector failures are correctness failures.
- Flaky tests are quality failures, not CI annoyances.
- Benchmark evidence must use stable fixtures and include allocation results when memory matters.
- pprof/trace evidence is required before performance-driven rewrites in critical code.
- Document unavailable tools instead of pretending they passed.
- A phase with non-trivial Go code should not score 90+ without static analysis evidence.

## 40. Persistence

## Recommendation

Persistence is an adapter, not the domain.

Database shape and domain shape may be related, but they are not automatically the same thing.

## Always do

- Keep persistence details outside domain logic.
- Make transaction boundaries explicit.
- Use repository/port abstractions where appropriate.
- Test custom queries.
- Test important mappings.
- Handle concurrency/locking deliberately.
- Use migrations for schema changes.
- Use parameterized queries.
- Avoid leaking database errors into domain APIs.
- Keep database models separate from domain models when invariants differ.
- Make persistence failure behavior explicit.

## Prefer

- Application services controlling transactions.
- Repository interfaces in application/domain boundary when useful.
- Infrastructure implementations using project-approved database package.
- sqlc, SQLBoiler, Ent, GORM, or database/sql only when project architecture accepts the tradeoffs.
- Testcontainers or integration database tests for non-trivial persistence.
- Explicit pagination.
- Optimistic concurrency/version fields where applicable.
- Migration tests where database schema changes are important.
- Separate read models when query shape differs from domain shape.

## Avoid

- Database rows as domain objects by accident.
- Business rules only in SQL.
- Hidden transactions.
- Long transactions.
- N+1 query patterns.
- Persistence code calling external services.
- Mapping code that silently drops fields.
- Starting transactions in handlers without application-level intent.
- Returning raw database errors through public APIs.

## Almost never do

- Store audit-critical records without traceability.
- Make transactions span slow external calls unless explicitly designed.
- Depend on production databases for tests.
- Use raw SQL for business-critical behavior without tests.
- Place legal/eSocial/SST decisions in query strings.


### Second-pass Go hardening

- Persistence is infrastructure; database rows are not automatically domain objects.
- Keep transaction boundaries explicit and controlled by application/use-case logic where practical.
- Do not hold database transactions open across slow external network/signing/XML operations unless designed and tested.
- Use parameterized queries or safe query builders.
- Test custom SQL, migrations, constraints, and important mappings.
- Distinguish not-found, conflict, validation, timeout, cancellation, and connection failures.
- Preserve context cancellation in database operations.
- Avoid leaking SQL driver errors directly through domain/public APIs.
- Test optimistic locking/version behavior when concurrent updates matter.
- Record audit-critical persistence changes with traceability.

## 41. API and Transport Layers

## Recommendation

APIs are contracts.

HTTP, GraphQL, gRPC, CLI, queue, and file interfaces must be explicit boundaries.

## Always do

- Use DTOs at external boundaries.
- Validate input.
- Return consistent errors.
- Avoid exposing internal error/debug details.
- Avoid exposing persistence records.
- Test serialization/deserialization.
- Version public APIs when breaking changes are introduced.
- Keep handlers thin.
- Keep use-case logic outside handlers.
- Keep framework-specific types out of domain logic.
- Make idempotency explicit for retryable operations.

## Prefer

- Clear request/response structs.
- Stable field names.
- Contract tests.
- Backward-compatible changes.
- Explicit idempotency for retryable operations.
- Problem Details or project-approved error format.
- Mappers from DTOs to domain/application types.
- User-safe error messages.
- Internal trace IDs for support.
- Dedicated validation layer before use-case execution.

## Avoid

- Framework request/context types flowing directly into domain logic.
- Domain types serialized directly by default.
- Transport errors replacing domain errors too early.
- Ambiguous status codes.
- Silent acceptance of malformed payloads.
- Business logic inside route handlers.
- Exposing raw %+v/debug output in API errors.
- Treating CLI parsing as business validation.

## Almost never do

- Break public contracts without tests and documentation.
- Expose stack traces or SQL errors.
- Put legal/eSocial/SST decisions inside HTTP handlers.
- Make API behavior depend on undocumented encoding defaults.
- Treat external API shape as internal domain shape.


### Second-pass Go hardening

- APIs are contracts; handlers should be thin and use cases should hold business workflow.
- Validate boundary input before converting to domain types.
- Map domain/application errors to stable transport errors deliberately.
- Do not expose stack traces, SQL errors, provider internals, or raw debug output to clients.
- Use DTOs for request/response bodies.
- Test JSON/XML field names, missing fields, invalid fields, unknown fields, null values, and status codes.
- Make idempotency explicit for retryable operations.
- Use request context for cancellation/deadlines and pass it through blocking operations.
- Avoid global mutable router/middleware state that tests cannot isolate.
- Contract-test public endpoints and CLI flags when they are part of a stable interface.

## 42. External Integrations

## Recommendation

External integrations must be isolated, contract-tested, timeout-bounded, retry-aware, and failure-aware.

## Always do

- Keep clients outside domain logic.
- Define explicit request/response models.
- Set timeouts.
- Handle retries deliberately.
- Make idempotency explicit.
- Validate responses before using them.
- Convert provider failures into application-level outcomes.
- Add contract tests or fake-server tests where applicable.
- Test error responses.
- Preserve correlation IDs.
- Keep credentials out of tests and logs.
- Test timeouts and malformed responses where practical.

## Prefer

- Interfaces/ports for external services.
- Provider-specific adapters.
- Versioned integration packages.
- httptest fake servers for HTTP integrations.
- Contract tests for:
  - payload shape
  - headers
  - status codes
  - success responses
  - error responses
  - timeout behavior
- Explicit retry/backoff packages only when justified.
- Circuit breakers only when operationally required and understood.

## Avoid

- Calling external services from domain logic.
- Retrying non-idempotent operations blindly.
- Infinite retries.
- No timeout HTTP clients.
- Using http.DefaultClient in production integrations without project policy.
- Ignoring malformed provider responses.
- Provider models leaking into domain logic.
- Live external systems in normal automated tests.

## Almost never do

- Send legal/eSocial/SST events without timeout/retry/audit design.
- Treat provider success as domain success without validation.
- Hide integration failure by logging and continuing.
- Store credentials in fixtures.
- Depend on production external services for phase verification.


### Second-pass Go hardening

- External clients must be adapter code, not domain code.
- Set timeouts on HTTP clients, requests, database calls, queues, and signing providers as applicable.
- Use fake servers/clients for tests instead of live providers.
- Test success, provider errors, malformed responses, timeouts, cancellation, retries, and idempotency.
- Preserve correlation IDs and external identifiers.
- Validate provider responses before using them for business decisions.
- Do not retry non-idempotent operations without idempotency keys or duplicate detection.
- Do not log credentials, tokens, raw sensitive payloads, or full provider responses without redaction.
- Version integration modules when provider contracts differ.
- Document fallback/degradation behavior explicitly.

## 43. eSocial/SST Strict Gate

## Recommendation

eSocial/SST code is legal/regulatory code. It must be treated as critical-risk by default.

Correctness is not only successful compilation. Correctness includes legal validity, schema/version correctness, deterministic XML, signing correctness, traceability, rejection handling, auditability, and supportability.

## Always do

- Keep eSocial/SST business rules outside handlers, database adapters, XML builders, and transport DTOs.
- Model event types and legal states explicitly.
- Keep event versions explicit.
- Validate required fields before XML generation.
- Validate domain rules before signing/transmission.
- Use version-specific XML models.
- Generate deterministic XML.
- Use golden tests for generated XML.
- Validate against schemas where available.
- Test rejection codes and business failure paths.
- Preserve receipts/protocols/rejection reasons.
- Redact sensitive XML/logs.
- Record audit trail for generation, signing, sending, rejection, correction, and cancellation.
- Test cancellation/correction state transitions.
- Treat certificate/signature handling as security-sensitive.

## Prefer

- Dedicated packages for:
  - event domain rules
  - event lifecycle
  - XML DTOs
  - XML mapping
  - schema validation
  - signing
  - transmission
  - receipt interpretation
  - audit
- Strong domain types for identifiers, versions, dates, periods, receipts, and legal codes.
- Fixture matrices for event versions and edge cases.
- Contract tests against known accepted/rejected payload examples.
- Fixed clocks for deadline/period rules.
- Explicit tenant/company/certificate boundaries.
- Hashes of generated payloads for traceability.

## Avoid

- Treating eSocial/SST payloads as generic maps.
- String concatenation for XML.
- Business rules hidden in XML tags or marshaling code.
- Domain models shaped by generated XML structs.
- Sending events without validating version/layout.
- Logging sensitive worker/health/personnel data.
- Making legal behavior depend on build tags.
- Retrying sends without idempotency and audit policy.

## Almost never do

- Accept eSocial/SST code without golden tests.
- Accept legal rule changes without regression tests.
- Accept signing code without deterministic canonicalization evidence.
- Accept transmission code without timeout, retry, and audit evidence.
- Accept receipt/rejection parsing without error-path tests.
- Score critical eSocial/SST code 90+ without coverage and mutation/fuzz evidence or documented mutation/fuzz readiness.


### Second-pass Go hardening

- eSocial/SST logic is critical-risk by default.
- Legal rules must be explicit, versioned, tested, and traceable.
- Keep event versions, layout versions, legal codes, receipt/protocol identifiers, and rejection codes strongly modeled.
- Do not represent legal event state as arbitrary strings or generic maps.
- Do not hide legal validation in XML tags, serializers, SQL queries, or HTTP handlers.
- Generated XML requires golden tests, schema validation where available, namespace tests, and deterministic ordering.
- Signing and transmission require correlation between payload, signature, certificate, request, response, receipt, and final state.
- Cancellation/correction flows require state-transition tests and audit tests.
- Sensitive personnel/health/payroll data must be redacted in logs and handled according to project policy.
- A phase touching eSocial/SST cannot score high without critical-path evidence.

## 44. Cryptography, Certificates, and Signing

## Recommendation

Cryptography must use maintained, reviewed libraries and explicit policies.

Certificate and signing code is security-critical and audit-critical.

## Always do

- Use Go standard library crypto packages or project-approved maintained libraries.
- Avoid custom cryptography.
- Validate certificates and chains according to project/legal requirements.
- Protect private keys.
- Avoid logging key material.
- Keep signing adapters isolated.
- Test success and failure paths.
- Test invalid/expired/wrong certificates.
- Test canonicalization/digest behavior where signatures require it.
- Record signing audit evidence where appropriate.
- Use constant-time comparison where secrets are compared.

## Prefer

- crypto/x509, crypto/tls, crypto/rsa, crypto/ecdsa, crypto/sha256, and project-approved algorithms.
- Explicit certificate store/loading policy.
- Secret manager integration instead of local files where applicable.
- Redacted types for secret-bearing values.
- Deterministic fixtures for signature tests that do not expose real private keys.
- Separate signing package from business rules and XML building.
- Rotation and expiration tests for certificate lifecycle.

## Avoid

- Hardcoded keys or certificates.
- Disabling certificate validation.
- Weak hashes/signatures for security-sensitive purposes.
- Homegrown XML signature implementation without strong justification.
- Private keys in test fixtures unless they are clearly fake and documented.
- Leaking crypto errors directly to external users when sensitive.

## Almost never do

- Implement custom cryptographic primitives.
- Store production private keys in source control.
- Sign legal XML without canonicalization tests.
- Treat certificate validation as optional.
- Suppress crypto/security tool findings without documented justification.


### Second-pass Go hardening

- Do not implement custom cryptography.
- Use standard library or project-approved cryptographic libraries.
- Validate certificate chains, expiration, key usage, revocation policy, and tenant/account ownership where required.
- Protect private keys and certificate material from logs, errors, fixtures, and source control.
- Make signing inputs deterministic and auditable.
- Test canonicalization, digest calculation, signature generation, signature verification, invalid certificate, expired certificate, wrong certificate, malformed payload, and provider rejection.
- Keep cryptographic adapters outside domain rules while preserving domain-level signing state.
- Avoid global mutable crypto configuration.
- Document algorithm, key storage, certificate source, rotation expectation, and failure behavior.
- Run vulnerability checks after crypto/signing dependency changes.

## 45. Configuration

## Recommendation

Configuration is an input boundary and must be validated.

Go services often read environment variables, files, flags, and secrets. Treat all of them as untrusted until validated.

## Always do

- Validate configuration at startup.
- Fail fast on missing required configuration.
- Redact secrets in logs/errors.
- Keep defaults explicit.
- Document required variables/flags.
- Test configuration parsing.
- Distinguish development defaults from production defaults.
- Keep configuration out of domain logic.
- Avoid reading environment variables deep inside business code.

## Prefer

- A dedicated config package.
- Typed configuration structs.
- Explicit loaders for env/files/flags/secrets.
- Validation methods returning structured errors.
- Safe duration/URL/path parsing.
- Startup evidence that configuration was validated without logging sensitive values.
- Dependency injection of validated config into adapters.

## Avoid

- Hidden os.Getenv calls across packages.
- Silent defaults for critical values.
- Logging complete config structs.
- Global mutable config.
- Parsing durations/URLs repeatedly.
- Reading config after startup without need.
- Treating CLI parsing as business validation.

## Almost never do

- Continue startup with invalid critical config.
- Store secrets in source control.
- Use production secrets in tests.
- Make legal/security behavior depend on undocumented env variables.
- Hide missing config by falling back to insecure defaults.


### Second-pass Go hardening

- Configuration is untrusted input until validated.
- Parse configuration at startup/composition boundaries and pass typed configuration to components.
- Avoid package-level reads from environment variables deep in domain/application code.
- Validate required fields, ranges, URLs, durations, paths, feature flags, and secret references.
- Distinguish missing, defaulted, and explicitly configured values.
- Do not log secrets when dumping configuration.
- Avoid global mutable config that tests cannot isolate.
- Test defaults, missing required config, invalid values, and secret redaction.
- Document environment variables and operational assumptions.
- Do not make business behavior depend on undocumented config flags.

## 46. Performance

## Recommendation

Performance work must be measured.

Go is fast enough for many services by default, but performance bugs still happen through allocation churn, goroutine leaks, reflection, excessive marshaling, inefficient queries, and unbounded concurrency.

## Always do

- Measure before optimizing non-trivial code.
- Keep correctness first.
- Benchmark performance-sensitive changes.
- Use realistic inputs.
- Document benchmark results when performance claims are made.
- Avoid premature unsafe/cgo optimization.
- Watch allocations where payload sizes are large.
- Test limits and timeouts.

## Prefer

- go test -bench with -benchmem for benchmarks.
- pprof for CPU/memory investigations.
- trace where goroutine scheduling/blocking matters.
- Preallocation when size is known and measured useful.
- Streaming parsers/writers for large payloads.
- Bounded concurrency.
- Simple algorithms before micro-optimizations.

## Avoid

- Optimizing without evidence.
- Using unsafe for speed without benchmark proof.
- Reflection-heavy hot paths.
- Excessive JSON/XML marshal/unmarshal cycles.
- Repeated large []byte/string conversions.
- Unbounded goroutines or queues.
- Caching without invalidation policy.

## Almost never do

- Sacrifice legal/security correctness for speed.
- Claim performance improvement without benchmark evidence.
- Use global caches for business-critical data without lifecycle/invalidations tests.
- Hide failed operations to improve apparent performance.


### Second-pass Go hardening

- Optimize only after correctness, tests, and evidence.
- Use benchmarks with representative inputs for performance-sensitive code.
- Record allocations with `-benchmem` when memory matters.
- Use pprof/trace evidence before complex rewrites.
- Avoid premature goroutines, pooling, unsafe, reflection, or custom parsers for imagined performance.
- Avoid unbounded memory growth from large slices, maps, caches, or response bodies.
- Define acceptable latency, throughput, and memory targets for performance work.
- Test large payloads and boundary sizes for parsers/generators.
- Keep performance optimizations readable and documented.
- Do not trade legal/security correctness for speed.

## 47. Resource Management

## Recommendation

Files, network bodies, database rows, transactions, timers, goroutines, and channels must be closed or released deliberately.

## Always do

- Close files and response bodies.
- Close database rows.
- Roll back or commit transactions explicitly.
- Stop timers/tickers when no longer needed.
- Cancel contexts when required.
- Drain or close channels according to ownership.
- Shut down workers gracefully.
- Test cleanup paths.
- Avoid leaks in error paths.

## Prefer

- defer for local cleanup when scope is clear.
- t.Cleanup in tests.
- context-aware APIs.
- Explicit Close methods for long-lived resources.
- Graceful shutdown tests for services/workers.
- Leak detection tools when project-approved.
- bounded readers for untrusted input.

## Avoid

- Leaking HTTP response bodies.
- Leaking database rows.
- Starting goroutines without shutdown.
- Starting tickers without stopping them.
- Defers inside large loops when they delay cleanup too long.
- Ignoring cleanup errors where they matter.
- Keeping files/network connections open across slow unrelated work.

## Almost never do

- Depend on garbage collection for external resource cleanup.
- Ignore transaction rollback/commit errors in critical paths.
- Leak goroutines in request-scoped workflows.
- Leave unbounded readers on untrusted input.
- Use finalizers for business-critical cleanup.


### Second-pass Go hardening

- Close files, response bodies, rows, statements, transactions, tickers, timers, and subscriptions deliberately.
- Check close/flush/commit errors when they affect correctness or durability.
- Ensure HTTP response bodies are closed and drained when connection reuse matters.
- Stop tickers and timers when no longer needed.
- Avoid goroutine leaks in tests and production code.
- Bound queues, caches, buffers, request bodies, and worker pools.
- Use context cancellation to release resources on timeout/shutdown.
- Avoid defers in tight loops when they delay resource release too long.
- Test cleanup on success, failure, cancellation, and panic/recover paths where relevant.
- Document ownership of resources passed across packages.

## 48. go generate and Generated Code

## Recommendation

Generated code must be deterministic, reviewable, and verified.

Generation is not a substitute for architecture.

## Always do

- Document every go generate step.
- Keep generator versions pinned by project policy.
- Ensure generated code builds and tests.
- Ensure generation produces no unintended diff.
- Mark generated files with the standard generated-code header.
- Keep generated code out of domain business rules unless generated from an authoritative legal/schema source and reviewed.
- Test generated boundaries.

## Prefer

- Generated DTOs/adapters only at boundaries.
- Golden tests for generated XML/JSON payloads.
- Deterministic sorting in generators.
- Separate packages for generated code.
- Human-authored domain models mapped from generated transport/schema models.

## Avoid

- Hidden generation during normal build.
- Generators that fetch network resources without explicit command and documentation.
- Generated code with local absolute paths.
- Generated code that changes due to timestamps.
- Modifying generated code manually unless project policy allows it.

## Almost never do

- Hide business logic in generators.
- Generate legal behavior without review and tests.
- Use generated transport structs as domain objects by accident.
- Claim a phase complete when generated code was not regenerated/verified.


### Second-pass Go hardening

- `go generate` is not run automatically by `go build`; required generation steps must be documented.
- Generated files must be deterministic and committed when project policy requires committed generated code.
- Do not require network access during generation unless the project explicitly approves and pins inputs.
- Pin generator versions or document how they are selected.
- Keep generated file headers, package names, build tags, and imports stable.
- Validate generated code with gofmt, go test, vet/static analysis, and contract/golden tests.
- Do not hand-edit generated code unless the file is no longer treated as generated.
- Do not hide business rules inside templates or generators without reviewable source and tests.
- Regenerate artifacts after changing schemas/protobufs/OpenAPI/XML definitions.
- Include deterministic diff evidence when generation is part of the phase.

## 49. Cross-Compilation, WASM, and Platform Considerations

## Recommendation

Go cross-compilation is powerful but platform differences still matter.

When platform behavior matters, test it or document the limitation.

## Always do

- Document target GOOS/GOARCH.
- Document CGO_ENABLED assumptions.
- Keep platform-specific files behind correct build tags.
- Test intended target builds where practical.
- Avoid hidden OS-specific path, signal, filesystem, or timezone assumptions.
- Keep binary/runtime dependencies explicit.

## Prefer

- Pure-Go implementations when portability matters.
- Separate platform adapter packages.
- Cross-build checks for deployable binaries.
- Integration tests on target-like environments for critical behavior.
- Explicit file permission handling.
- Explicit signal/shutdown handling for services.

## Avoid

- Assuming Linux behavior on Windows/macOS.
- Assuming local timezone/locale.
- Using cgo when static/container deployment requires CGO_ENABLED=0.
- Platform-specific syscalls in business packages.
- Build tags that are not tested.

## Almost never do

- Introduce cgo into portable code without documenting deployment consequences.
- Claim cross-platform support without target build evidence.
- Put platform-specific behavior in domain logic.
- Hide platform behavior behind undocumented build tags.


### Second-pass Go hardening

- Cross-compilation must be verified for supported `GOOS`/`GOARCH` targets affected by the phase.
- cgo can prevent simple cross-compilation; document platform requirements.
- Platform-specific files must have clear build tags and tests where practical.
- Do not assume filesystem paths, case sensitivity, line endings, signals, process behavior, or permissions are identical across platforms.
- WASM/tiny/container targets may restrict networking, filesystem, time, signals, or cgo.
- Verify target-specific serialization, path handling, and certificate/root-store assumptions when relevant.
- Keep platform-specific business behavior out of domain logic.
- Document unsupported platforms explicitly.
- Avoid accidental dependence on host platform in tests.
- Use CI matrix evidence where available for multi-platform code.

## 50. Regression Tests

## Recommendation

Every bug fix must add a regression test unless there is a documented reason.

Regression tests preserve learned knowledge.

## Always do

- Add a test that fails before the fix and passes after it when practical.
- Name regression tests by behavior, not ticket number only.
- Include edge cases that caused the failure.
- Keep regression fixtures minimal and understandable.
- Add golden/corpus fixtures for parser/XML/serialization bugs.
- Document any missing regression test in PHASE-RESULT.md.

## Prefer

- Unit regression tests for business rules.
- Integration regression tests for adapter failures.
- Fuzz corpus additions for parser bugs.
- Golden files for payload regressions.
- Race tests for concurrency regressions.
- Table tests when one bug reveals a rule family.

## Avoid

- Fixing bugs only by changing implementation.
- Tests that merely assert no panic when behavior matters.
- Regression tests that depend on real external services.
- Broad snapshots that obscure the exact regression.

## Almost never do

- Close a legal/eSocial/SST bug without a regression test.
- Remove a failing regression test to finish a phase.
- Claim a bug is fixed without evidence.
- Leave concurrency bugs covered only by sleeps.


### Second-pass Go hardening

- Every fixed bug should add a regression test that fails without the fix.
- Regression tests should encode the observed failure, not only a broad happy path.
- Include provider payloads, XML fixtures, dates, numeric values, and edge inputs that caused the bug when safe.
- Redact sensitive real-world data before turning it into fixtures.
- Name regression tests after behavior, not ticket numbers only.
- Keep regression tests deterministic.
- Avoid deleting regression tests because they are inconvenient.
- Add regression corpus entries for fuzz-discovered cases.
- Add audit/security regression tests when the bug affected traceability or exposure.
- Document unfixed related risks in PHASE-RESULT.md.

## 51. LLM-Specific Go Anti-Patterns

## Recommendation

Generated Go often looks plausible while hiding serious quality problems.

The implementation LLM must actively avoid known Go generation failures.

## Always do

- Verify code compiles and tests pass.
- Check every error return.
- Avoid nil pointer risks.
- Avoid global mutable state.
- Avoid unbounded goroutines.
- Avoid context misuse.
- Avoid package cycles.
- Avoid fake abstractions.
- Avoid business rules in handlers.
- Avoid silent zero values.
- Avoid generated tests with weak assertions.

## Prefer

- Simple packages with clear responsibilities.
- Explicit constructors and validation.
- Narrow interfaces at boundaries.
- Concrete types in domain code.
- Table tests with meaningful assertions.
- Deterministic fixtures.
- Small dependency additions with documented purpose.

## Avoid

- Ignoring errors with _.
- Returning nil, nil.
- panic("not implemented").
- TODO comments in completed phase code.
- fmt.Println debugging.
- Reflection to avoid types.
- map[string]any domain models.
- Overusing interfaces because “Go likes interfaces.”
- Overusing goroutines because “Go is concurrent.”
- Tests that only call functions and do not assert results.

## Almost never do

- Claim completion without command evidence.
- Invent APIs from dependencies without verifying usage.
- Add dependency imports that are not used.
- Use code that only works in the LLM’s imagined project structure.
- Produce legal/regulatory logic without fixtures, edge cases, and audit behavior.


### Second-pass Go hardening

- Do not use `context.TODO()` everywhere because the LLM did not know the caller context.
- Do not ignore errors with `_` or comments like "cannot happen".
- Do not create interfaces for every dependency just to look testable.
- Do not put all code in `main.go`, `service.go`, `handler.go`, or `utils.go`.
- Do not return `map[string]any` for domain outcomes.
- Do not use goroutines for sequential work to look idiomatic.
- Do not use channels where a simple function return is clearer.
- Do not use reflection because field names were easier than modeling types.
- Do not use `panic` as placeholder control flow.
- Do not claim a phase is done when only generated boilerplate exists.

## Extra LLM review prompts

- Did I invent a package/API name that does not exist?
- Did I assume a dependency’s API without checking the project version?
- Did I add a dependency to compensate for not understanding the standard library?
- Did I produce tests that merely mirror implementation logic?
- Did I hide uncertainty behind vague comments?

## 52. Recommended Tooling Matrix

| Purpose | Baseline Tool | Stronger/Optional Tooling |
|---|---|---|
| Format | gofmt | goimports, golangci-lint fmt when project-approved |
| Compile/test | go test ./... | go test -count=1, go test -json |
| Suspicious code | go vet | staticcheck, golangci-lint |
| Race detection | go test -race | realistic race-built workload |
| Coverage | go test -coverprofile | go tool cover, CI coverage reporting |
| Fuzzing | go test -fuzz | scheduled long fuzz runs, saved corpus |
| Vulnerabilities | govulncheck | gosec, supply-chain policy checks |
| Dependencies | go mod tidy, go list -m all | go mod why, license checks |
| Performance | go test -bench -benchmem | pprof, trace |
| Generated code | go generate | deterministic diff checks |
| Architecture | package/internal structure | import restriction linters, depguard |


### Second-pass Go hardening

- Tooling choice must follow project policy first; this matrix is a baseline, not permission to add tools casually.
- Prefer tools that can run locally and in CI with the same configuration.
- Pin tool versions where reproducibility matters.
- Document unavailable tools in PHASE-RESULT.md.
- Do not install or run tools that fetch arbitrary network resources in restricted environments without approval.
- Do not treat optional tools as required if the project has not approved them, but do document the risk when a useful tool is absent.
- For critical phases, stronger evidence should be added even if not previously configured.
- Use tool output to improve code, not merely to fill a checklist.
- Keep tool configuration small, readable, and reviewed.
- Avoid new tool dependencies that create more risk than they reduce.

## Expanded tool candidates

| Purpose | Candidate tools/commands | Notes |
|---|---|---|
| Format/imports | `gofmt`, `goimports` | `gofmt` is mandatory; `goimports` is project-policy dependent. |
| Compile/test | `go test`, `go test -run=^$` | Full tests are stronger than compile-only checks. |
| Vet/static | `go vet`, `staticcheck`, `golangci-lint` | Suppress narrowly and document. |
| Error checks | `errcheck` via configured linter | Critical for persistence, files, transactions, and HTTP. |
| Race | `go test -race` | Required for concurrency-sensitive changes where supported. |
| Coverage | `go test -coverprofile`, `go tool cover` | Use risk-based thresholds. |
| Fuzz | `go test -fuzz` | Best for parsers/validators/untrusted input. |
| Mutation | `gremlins`, project-approved mutation tools | Focus on business rules first. |
| Vulnerabilities | `govulncheck` | Strong signal for reachable vulnerable dependency use. |
| Security lint | `gosec` when configured | Useful but can be noisy; review findings. |
| Dependencies | `go mod tidy`, `go list -m all`, `go mod why -m` | Explain graph changes. |
| Performance | `go test -bench -benchmem`, `pprof`, `trace` | Require representative fixtures. |
| Architecture | `go list`, depguard/import restriction linters | Enforce package direction. |

## 53. Architecture Test Ideas

## Recommendation

Architecture should be tested where language/package boundaries are not enough.

## Always do

- Test business rules directly in domain/application packages.
- Test DTO-to-domain mapping at boundaries.
- Test infrastructure adapters separately from domain rules.
- Verify handlers call use cases rather than embedding workflows.
- Verify generated XML/JSON does not become the domain model.

## Prefer

- Import restriction lint rules.
- depguard or custom go list checks.
- Tests that ensure domain packages do not import infrastructure/framework packages.
- Tests that ensure API DTO packages do not leak into domain packages.
- Tests that ensure persistence records are mapped explicitly.
- Compile-time interface assertions for adapters.

## Avoid

- Relying only on code review for critical boundaries.
- Letting test packages import internals in ways production code cannot.
- Circular package dependencies solved through common dumping grounds.
- Architecture checks that are too brittle to maintain.

## Almost never do

- Let handlers/database/XML packages become the legal rules engine.
- Bypass internal package boundaries through copy-paste.
- Move code only to satisfy import cycles without fixing design.
- Accept high quality score with obvious boundary violations.


### Second-pass Go hardening

- Architecture tests should be automated when boundary drift is likely.
- Use `go list -deps -json` or configured linters to detect forbidden imports.
- Domain packages should fail checks if they import transport, database, XML adapter, logging implementation, or provider packages.
- API packages should fail checks if they bypass application use cases for business workflows.
- Persistence packages should fail checks if they import HTTP/router packages.
- Generated payload packages should not be imported by domain packages unless explicitly approved.
- Use compile-time interface assertions for adapters implementing application ports.
- Add package-level tests for mapping behavior at every boundary.
- Keep architecture tests simple enough that maintainers trust them.
- Document intentionally allowed exceptions.

## 54. Definition of Done

A Go phase is done only when all applicable items are true:

1. Planned implementation exists.
2. Code is in the expected packages/modules.
3. Code builds.
4. gofmt check passes.
5. go vet passes or justified exceptions are documented.
6. Configured linters pass or justified exceptions are documented.
7. Meaningful automated tests exist.
8. Tests pass.
9. Race detector was run for concurrency-sensitive code where practical.
10. Coverage was measured where required.
11. Mutation/fuzz/property evidence exists for critical rules where practical.
12. Dependency changes are minimal and documented.
13. govulncheck/security checks were run where available for meaningful dependency/security changes.
14. Architectural boundaries are preserved.
15. Errors are explicit and tested.
16. Panics are absent or justified.
17. unsafe/cgo is absent or justified.
18. Context/cancellation behavior is deliberate.
19. Logging avoids secrets and sensitive payloads.
20. Auditability exists where required.
21. PHASE-RESULT.md exists and contains command evidence.
22. Quality score is supported by evidence.


### Second-pass Go hardening

- Definition of done must be stricter as risk increases.
- Low-risk done means clean build/format/vet/tests for existing behavior.
- Medium-risk done means tested success, failure, boundary, and integration seams.
- High-risk done means coverage, regression, edge cases, and architecture evidence.
- Critical-risk done means golden/contract/audit/security/fuzz-or-mutation-readiness evidence.
- Done requires PHASE-RESULT.md to match actual commands and outcomes.
- Done requires no hidden local assumptions.
- Done requires no unexplained dependency/toolchain changes.
- Done requires any residual risk to be explicit.
- Done is not a feeling; it is evidence.

## 55. PHASE-RESULT.md Required Template

PHASE-RESULT.md must include at least:

```markdown
# PHASE-RESULT.md

## Summary

- Phase:
- Scope completed:
- Scope not completed:

## Toolchain

- Go version:
- Relevant go env:
- Module/workspace:
- GOOS/GOARCH:
- CGO_ENABLED:

## Files changed

- path:
  - reason:

## Architecture

- Boundaries preserved:
- Boundary deviations:
- Reason for deviations:

## Dependencies

- Added:
- Removed:
- Updated:
- Reason:
- go.mod/go.sum impact:

## Commands run

- _**`command here`**_

## Commands passed

- _**`command here`**_

## Commands failed

- _**`command here`**_
  - Reason:
  - Impact:
  - Required fix:

## Tests

- Unit tests:
- Integration tests:
- Golden tests:
- Fuzz/property tests:
- Regression tests:
- Race detector:

## Coverage

- Tool:
- Line coverage:
- Critical uncovered paths:
- Reason:

## Mutation/Fuzz Evidence

- Tool/command:
- Result:
- Surviving mutants or failures:
- Required follow-up:

## Security and Supply Chain

- govulncheck:
- gosec or configured security tool:
- License/dependency checks:
- Findings:

## Panic / unsafe / cgo Review

- Panics introduced:
- unsafe introduced:
- cgo introduced:
- Justification:
- Tests/evidence:

## Observability and Audit

- Logs/traces:
- Redaction:
- Audit records:
- Correlation IDs:

## Quality Score

- Score:
- Evidence supporting score:
- Risks remaining:
```


### Second-pass Go hardening

- PHASE-RESULT.md must be specific enough that another engineer can reproduce the evidence.
- Include exact command strings, working directory, pass/fail outcome, and important environment values.
- Include test categories, not only aggregate pass/fail.
- Include coverage numbers when required and explain uncovered critical paths.
- Include dependency changes, new modules, removed modules, replacements, and vulnerability results.
- Include architecture deviations and why they were necessary.
- Include panic/unsafe/cgo review even if the result is "none introduced".
- Include concurrency/race/cancellation evidence when relevant.
- Include audit/redaction/security evidence for critical flows.
- Include a risk-based score rationale instead of a naked number.

## Minimum PHASE-RESULT.md quality bar

A good PHASE-RESULT.md lets a reviewer answer:

- What changed?
- How was it verified?
- What failed and was fixed?
- What could not be verified?
- What risks remain?
- Why is the score justified?

## 56. Quality Score Model

The quality score must be evidence-based.

## Scoring bands

| Score | Meaning |
|---:|---|
| 0-39 | Not acceptable. Missing build/test/evidence or severe design flaws. |
| 40-59 | Weak. Some evidence exists, but important quality gates are missing. |
| 60-74 | Basic acceptable for low-risk code only. Gaps must be documented. |
| 75-84 | Good. Most relevant gates pass with meaningful tests. |
| 85-89 | Strong. Good evidence, boundary discipline, failure tests, and security checks. |
| 90-94 | Very strong. Suitable for high-risk code with coverage, static analysis, and strong tests. |
| 95-100 | Exceptional. Critical-path evidence, mutation/fuzz/contract tests, audit/security proof, minimal risk. |

## Score constraints

- Maximum 49 if the code does not build.
- Maximum 59 if gofmt was not checked.
- Maximum 64 if tests were not run.
- Maximum 69 if meaningful tests are missing for implemented behavior.
- Maximum 74 if go vet/static checks were skipped without reason.
- Maximum 79 if dependency/security checks were skipped after meaningful dependency/security changes.
- Maximum 84 if concurrency-sensitive code lacks race/cancellation evidence.
- Maximum 84 if error paths are not tested in medium/high-risk code.
- Maximum 89 if critical legal/security/signing code lacks golden/contract/error-path tests.
- Maximum 89 if coverage requirements were skipped without reason.
- Maximum 89 if unsafe/cgo is present without strong tests and documentation.
- Maximum 94 if critical code lacks mutation/fuzz evidence or documented mutation/fuzz readiness.

## Required scoring evidence

- Commands actually run.
- Pass/fail status.
- Test names or categories.
- Coverage results where required.
- Security/dependency results where required.
- Explicit residual risks.
- Justified exceptions.


### Second-pass Go hardening

- The score must never reward volume of generated code.
- The score must penalize missing evidence even when the code looks plausible.
- A high score requires relevant tests, not just many tests.
- A high score requires failure-path evidence for non-trivial behavior.
- A high score requires dependency/security evidence after dependency changes.
- A high score requires race/cancellation evidence for concurrency-sensitive code.
- A high score requires golden/contract/audit evidence for legal/eSocial/SST/signing flows.
- A high score requires documented tool failures or unavailable tools.
- Do not score above 74 when the reviewer cannot reproduce the verification commands.
- Do not score above 89 for critical code without strong edge-case and negative-case tests.

## 57. Caveman Quality Review

Before completion, perform a blunt review.

Ask:

- Does it build?
- Did I run the commands?
- Did I prove it with tests?
- Did I check errors?
- Did I avoid panic?
- Did I avoid nil traps?
- Did I avoid goroutine leaks?
- Did I avoid races?
- Did I avoid unsafe/cgo unless justified?
- Did I keep business rules out of handlers/database/XML?
- Did I validate inputs?
- Did I protect secrets?
- Did I document dependency changes?
- Did I preserve auditability?
- Did I create PHASE-RESULT.md?
- Can another engineer trust this without guessing?

If the answer to any relevant question is no, the phase is not complete.


### Second-pass Go hardening

- Would I trust this code during an incident?
- Would I trust this code with malformed input?
- Would I trust this code with a canceled request?
- Would I trust this code when two goroutines execute it at the same time?
- Would I trust this code after a dependency update?
- Would I trust this code if a provider returns nonsense?
- Would I trust this code if a legal deadline lands on a timezone boundary?
- Would I trust this code if the audit log is subpoenaed or used for support?
- Would I know exactly what command proved it works?
- Would another maintainer understand the design without asking the LLM?

## 58. Final Checklist

Before sending the final message, verify:

- [ ] Planned implementation exists.
- [ ] Code is formatted.
- [ ] Code builds.
- [ ] go vet ran.
- [ ] Configured linters ran.
- [ ] Tests exist.
- [ ] Tests pass.
- [ ] Error paths are tested.
- [ ] Edge cases are tested.
- [ ] Race detector ran where relevant/practical.
- [ ] Coverage was measured where required.
- [ ] Fuzz/property tests exist where useful.
- [ ] Mutation testing or mutation-readiness is documented for critical rules.
- [ ] Dependency changes are justified.
- [ ] go.mod/go.sum changes are intentional.
- [ ] govulncheck/security checks ran where relevant/available.
- [ ] No unreviewed panic/log.Fatal/os.Exit in reusable packages.
- [ ] No unreviewed unsafe/cgo.
- [ ] No unchecked critical errors.
- [ ] No sensitive logs.
- [ ] Auditability exists where required.
- [ ] Architecture boundaries are preserved.
- [ ] PHASE-RESULT.md exists.
- [ ] Quality score is evidence-based.
- [ ] Final response is exactly: I finished the implementation

### Second-pass Go hardening

- [ ] `go version` evidence captured.
- [ ] Relevant `go env` evidence captured.
- [ ] `gofmt` check produced no unformatted files.
- [ ] `go test ./...` passed or failures are documented with required fixes.
- [ ] `go vet ./...` passed or justified exceptions are documented.
- [ ] Configured static analysis passed or justified exceptions are documented.
- [ ] Race detector ran for concurrency-sensitive changes where supported/practical.
- [ ] `go mod tidy` was run or intentionally skipped with reason.
- [ ] `go.mod`/`go.sum` diffs are intentional.
- [ ] Dependency vulnerability checks ran when dependency/security changes occurred.
- [ ] Serialization contracts have tests when changed.
- [ ] Persistence changes have query/migration/mapping tests.
- [ ] External integrations have timeout/error/malformed-response tests.
- [ ] Audit-critical flows record success and failure events.
- [ ] Sensitive data redaction was reviewed.
- [ ] `panic`, `log.Fatal`, `os.Exit`, `unsafe`, and `cgo` were reviewed.
- [ ] Generated code is deterministic and reviewed.
- [ ] Build tags/platform targets affected by the phase were tested.
- [ ] PHASE-RESULT.md contains exact command evidence.
- [ ] The final response rule is followed exactly.
