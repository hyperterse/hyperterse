<!--
Thank you for contributing to Hyperterse!

Please fill out this template to help us review your pull request effectively.
If a section is not applicable, you can mark it as N/A or remove it.
-->

## Description

<!-- Provide a clear and concise description of your changes. -->

### Related Issues

<!-- Link to related issues using keywords: Fixes #123, Closes #456, Relates to #789 -->
<!-- REMOVE THIS SECTION IF NOT RELEVANT -->

## Type of Change

<!-- Check all that apply -->

- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] New feature (non-breaking change that adds functionality)
- [ ] Breaking change (fix or feature that causes existing functionality to change)
- [ ] Documentation update
- [ ] Configuration change
- [ ] Code refactoring (no functional changes)
- [ ] Performance improvement
- [ ] Test improvements
- [ ] Build/Infrastructure changes
- [ ] Security fix

## Breaking Changes

<!-- If this PR contains breaking changes, describe them here -->

- [ ] This PR introduces breaking changes
- [ ] Migration guide included (if applicable)

<details>
<summary>Breaking Changes Details</summary>

<!-- Describe what breaks and how users should update their code -->

</details>

## Testing

### Manual Testing

- [ ] Tested locally with development server (`hyperterse dev`)
- [ ] Tested with production build (`make build`)
- [ ] Tested error handling and edge cases
- [ ] Tested with sample configurations

### Tested Scenarios

<!-- Describe the scenarios you tested -->

1.
2.
3.

## Code Quality

### Go Standards

- [ ] Code follows Go conventions and best practices
- [ ] Code formatted with `go fmt ./...`
- [ ] No new linter warnings introduced
- [ ] Error handling is explicit and appropriate
- [ ] Exported functions/types have proper documentation
- [ ] No hardcoded values (use constants or configuration)

### Code Review

- [ ] Self-review completed
- [ ] Code is DRY (Don't Repeat Yourself)
- [ ] Complex logic includes explanatory comments
- [ ] No commented-out code left in PR
- [ ] No debug logging or console output left in code

## Documentation

- [ ] README.md updated (if user-facing changes)
- [ ] CONTRIBUTING.md updated (if development workflow changes)
- [ ] CHANGELOGS.md updated with changes
- [ ] Inline code documentation added/updated
- [ ] Documentation site updated (if applicable):
  - [ ] Configuration reference (`docs/src/content/docs/reference/configuration.mdx`)
  - [ ] CLI reference (`docs/src/content/docs/reference/cli.mdx`)
  - [ ] Guide added/updated (`docs/src/content/docs/guides/`)
  - [ ] Concepts updated (`docs/src/content/docs/concepts/`)
- [ ] API documentation generated correctly (OpenAPI/MCP)
- [ ] Examples updated or added

## Security

- [ ] No sensitive information (credentials, API keys) in code
- [ ] Input validation implemented for new inputs
- [ ] SQL injection prevention verified (parameterized queries)
- [ ] Error messages don't leak sensitive information
- [ ] Security best practices followed
- [ ] Dependencies are up to date and secure

## Performance

- [ ] No performance regressions introduced
- [ ] Efficient database queries (no N+1 queries)
- [ ] Appropriate use of caching (if applicable)
- [ ] Resource cleanup implemented (connections, goroutines)
- [ ] Benchmarks added/updated (if performance-critical)

<details>
<summary>Performance Metrics</summary>

<!-- Include benchmark results if applicable -->

```bash
# go test -bench=. -benchmem
```

</details>

## Component-Specific Checklist

<!-- Check relevant sections based on what your PR touches -->

### CLI Changes

- [ ] CLI commands work as expected
- [ ] Help text is accurate and helpful
- [ ] Flags/arguments are properly validated
- [ ] Exit codes are appropriate
- [ ] User-friendly error messages

### Parser Changes

- [ ] Valid configurations parse correctly
- [ ] Invalid configurations produce helpful error messages
- [ ] Schema validation works
- [ ] Backward compatibility maintained (if applicable)
- [ ] JSON schemas updated (`schema/root.terse.schema.json`, `schema/adapter.terse.schema.json`, `schema/tool.terse.schema.json`)
- [ ] Configuration schema docs updated (`docs/reference/configuration-schemas.mdx`)

### Connector Changes

- [ ] Connector interface properly implemented
- [ ] Connection pooling handled correctly
- [ ] Graceful error handling and recovery
- [ ] Connection cleanup on shutdown
- [ ] Database-specific features tested
- [ ] Connector registered in factory

### Runtime/Server Changes

- [ ] HTTP endpoints work correctly
- [ ] Error responses are properly formatted
- [ ] Logging is appropriate (not too verbose, not too sparse)
- [ ] Graceful shutdown implemented
- [ ] Concurrent requests handled safely

### MCP/OpenAPI Changes/Docs

- [ ] MCP protocol compliance verified
- [ ] OpenAPI spec generated correctly
- [ ] Tool definitions are accurate
- [ ] JSON-RPC responses follow specification
- [ ] `/docs` endpoint displays correctly
- [ ] `/llms.txt` endpoint displays correctly

## Distribution & Deployment

- [ ] Build script works (`make build`)
- [ ] Binary runs on target platforms
- [ ] Installation script works (if modified)
- [ ] Distribution packages build correctly:
  - [ ] Homebrew formula (if applicable)
  - [ ] NPM package (if applicable)

## Configuration & Compatibility

- [ ] Backward compatible with existing configurations
- [ ] Environment variable handling tested
- [ ] Configuration validation works
- [ ] Default values are sensible
- [ ] `CHANGELOGS.md` updated with breaking changes

## Additional Context

<!-- Add any additional context, screenshots, or examples -->

### Screenshots/Examples

<!-- If applicable, add screenshots or example output -->

### Dependencies

<!-- List any new dependencies added and justify their inclusion -->

- None / N/A

### Migration Guide

<!-- If this is a breaking change, provide migration instructions -->

<details>
<summary>Migration Steps</summary>

<!-- Step-by-step guide for users to migrate -->

</details>

## Reviewer Notes

<!-- Any specific areas you'd like reviewers to focus on? -->

---

<!--
By submitting this pull request, I confirm that:
- I have read the CONTRIBUTING.md guidelines
- I agree to the terms in the Code of Conduct
- My contribution is licensed under the project's license
-->
