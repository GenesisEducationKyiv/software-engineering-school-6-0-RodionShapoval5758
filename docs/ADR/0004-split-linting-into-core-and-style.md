# ADR-0004: Split linting into core and style configurations

## Context
The project uses `golangci-lint`, but not all linters have the same purpose. Some checks are high-signal correctness checks that should act as a quality gate, while others are more stylistic or readability-oriented and can create more review noise.

The project also uses CI, so linting policy needs a clear enforcement model instead of one mixed list of rules.

## Decision
Split linting into two separate `golangci-lint` configurations:
- `.golangci.core.yaml` for correctness-oriented checks
- `.golangci.style.yaml` for softer style and maintainability checks

Run them as separate CI jobs.

## Consequences
### Positive
- keeps correctness checks separate from lower-priority style rules
- makes CI policy clearer and easier to evolve
- reduces the risk that noisy style rules weaken trust in important lint failures
- allows style rules to be tuned independently without weakening the core quality gate

### Negative
- adds one more config file and one more CI job
- lint policy is slightly more complex than a single-file setup

### Tradeoffs
- more explicit and maintainable than a single mixed config, at the cost of a small amount of tooling complexity

## Alternatives Considered
### Single combined lint configuration
Rejected because it mixes blocking correctness checks with softer stylistic rules and makes enforcement policy less clear.

### Only one strict core configuration with no separate style checks
Rejected because style and maintainability checks are still useful, but they should not have the same weight as correctness-oriented checks.
