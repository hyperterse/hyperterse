---
globs:
alwaysApply: false
---

# Cursor Rules for Hyperterse

## Core Principles

### 1. Always Ask Clarifying Questions First

**Before implementing any feature or change:**

- **Understand the requirement**: Ask questions to clarify:

  - What is the exact problem being solved?
  - What is the expected behavior?
  - Are there edge cases to consider?
  - What are the constraints (performance, security, compatibility)?
  - Is this a breaking change or backward compatible?

- **Verify assumptions**: Don't assume:

  - Ask about user intent and use cases
  - Clarify ambiguous requirements
  - Confirm understanding before coding
  - Discuss trade-offs and alternatives

- **Example questions to ask**:
  - "What is the specific use case for this feature?"
  - "Should this be backward compatible?"
  - "Are there performance requirements?"
  - "What error handling is expected?"
  - "Should this be configurable or hardcoded?"

### 2. Verify Everything with Tests

**Testing is mandatory, not optional:**

- **Write tests first** (TDD when possible):

  - Unit tests for all new functions
  - Integration tests for new features
  - Edge case tests
  - Error case tests

- **Test coverage requirements**:

  - Aim for >80% coverage for new code
  - Test happy paths
  - Test error paths
  - Test edge cases (empty inputs, nil values, boundary conditions)
  - Test validation logic thoroughly

- **Test types to write**:

  - Unit tests: Test individual functions in isolation
  - Integration tests: Test component interactions
  - End-to-end tests: Test full request/response cycles
  - Validation tests: Test input validation logic
  - Connector tests: Test database interactions (with test databases)

- **Before merging**:
  - All tests must pass: `go test ./...`
  - No linting errors: `golangci-lint run`
  - Build succeeds: `go build`
  - Manual testing completed

### 3. Code Quality Standards

**Follow Go best practices:**

- **Formatting**: Always run `go fmt` before committing
- **Linting**: Fix all linting errors before submitting
- **Naming**: Use clear, descriptive names following Go conventions
- **Comments**: Document exported functions and complex logic
- **Error handling**: Always handle errors explicitly, never ignore them
- **Type safety**: Use strong types, avoid `interface{}` when possible

**Code organization:**

- Keep functions focused and small (single responsibility)
- Avoid deep nesting (max 3-4 levels)
- Use early returns to reduce nesting
- Group related functionality together
- Follow the existing package structure

### 4. Security First

**Security considerations:**

- **Never expose sensitive data**:

  - Connection strings never in responses or logs
  - SQL statements not in error messages (unless safe)
  - No stack traces in production errors

- **Input validation**:

  - Validate all inputs before use
  - Use type checking to prevent injection
  - Sanitize error messages
  - Escape SQL values properly

- **When adding new features**:
  - Consider security implications
  - Review for potential vulnerabilities
  - Test with malicious inputs
  - Document security assumptions

### 5. Documentation Standards

**Keep documentation updated:**

- **Code comments**:

  - Document exported functions
  - Explain complex algorithms
  - Add examples for public APIs
  - Document error conditions

- **Living documents**:

  - Update `HYPERTERSE.md` when adding features
  - Update `README.md` for user-facing changes
  - Keep examples current
  - Document breaking changes

- **When to update docs**:
  - New features added
  - API changes
  - Architecture changes
  - New connectors or types
  - Security considerations change

### 6. Architecture Consistency

**Maintain the existing structure:**

- **Package organization**:

  - `core/parser/`: Configuration parsing and validation
  - `core/runtime/`: Server, execution, connectors
  - `core/types/`: Type definitions (generated)
  - `core/logger/`: Logging utilities
  - `pkg/pb/`: Generated protobuf code (don't edit manually)

- **When adding code**:
  - Place code in the appropriate package
  - Follow existing patterns
  - Don't create new packages without discussion
  - Keep related functionality together

### 7. Error Handling

**Comprehensive error handling:**

- **Error messages**:

  - Be specific and actionable
  - Include context (what operation failed)
  - Don't expose sensitive information
  - Use structured errors when appropriate

- **Error types**:

  - Use custom error types for different error categories
  - Wrap errors with context: `fmt.Errorf("operation failed: %w", err)`
  - Return errors, don't panic (except in truly unrecoverable cases)

- **Validation errors**:
  - Collect all validation errors, don't stop at first
  - Format errors clearly for users
  - Provide field-level error information

### 8. Performance Considerations

**Keep performance in mind:**

- **Database connections**:

  - Use connection pooling (already configured)
  - Close connections properly
  - Don't leak connections

- **Query execution**:

  - Consider query performance
  - Use appropriate indexes (document in examples)
  - Avoid N+1 query problems

- **Memory**:
  - Stream large results when possible
  - Avoid loading entire datasets into memory
  - Clean up resources properly

### 9. Backward Compatibility

**Maintain compatibility:**

- **Breaking changes**:

  - Discuss before implementing
  - Provide migration path
  - Update version numbers
  - Document changes clearly

- **Configuration changes**:
  - Support old format when possible
  - Provide clear upgrade instructions
  - Deprecate gradually

### 10. Review Checklist

**Before submitting code:**

- [ ] Asked clarifying questions if requirements were unclear
- [ ] Written comprehensive tests (unit + integration)
- [ ] All tests pass: `go test ./...`
- [ ] No linting errors
- [ ] Code formatted: `go fmt ./...`
- [ ] Build succeeds: `go build`
- [ ] Documentation updated (if needed)
- [ ] Security considerations reviewed
- [ ] Error handling comprehensive
- [ ] Manual testing completed
- [ ] No sensitive data exposed
- [ ] Backward compatibility considered

## Specific Guidelines

### Adding New Connectors

1. **Ask**: What database? What are the connection requirements?
2. **Plan**: Review existing connector implementations
3. **Implement**: Follow the `Connector` interface pattern
4. **Test**: Write tests with test database
5. **Document**: Update `HYPERTERSE.md` with connector details

### Adding New Types

1. **Ask**: What is the use case? What validation is needed?
2. **Plan**: Review existing type implementations
3. **Implement**: Add to proto, regenerate, add conversion logic
4. **Test**: Write validation tests
5. **Document**: Update type documentation

### Modifying Parsers

1. **Ask**: What format? What syntax? Backward compatible?
2. **Plan**: Review existing parser patterns
3. **Implement**: Follow existing parser structure
4. **Test**: Write parser tests with various inputs
5. **Document**: Update configuration documentation

### Changing API

1. **Ask**: Breaking change? Migration path?
2. **Plan**: Review impact on existing users
3. **Implement**: Add versioning if needed
4. **Test**: Test all endpoints affected
5. **Document**: Update API documentation

## Questions to Ask Before Coding

### Feature Requests

- What problem does this solve?
- Who is the target user?
- What is the expected behavior?
- Are there edge cases?
- What are the performance requirements?
- Is this backward compatible?

### Bug Fixes

- What is the exact issue?
- Can it be reproduced?
- What is the expected behavior?
- What is the root cause?
- Are there similar issues?
- What is the impact?

### Refactoring

- What is the goal?
- What are the risks?
- Are tests in place?
- Is backward compatibility maintained?
- What is the migration path?

## Testing Requirements

### Unit Tests

- Test each function independently
- Mock dependencies
- Test happy paths
- Test error cases
- Test edge cases

### Integration Tests

- Test component interactions
- Use test databases
- Test full request/response cycles
- Test error propagation

### Validation Tests

- Test all validation rules
- Test error message formatting
- Test edge cases (empty, nil, invalid types)

### Connector Tests

- Test connection establishment
- Test query execution
- Test error handling
- Test connection cleanup
- Use test databases (don't use production)

## Common Mistakes to Avoid

1. **Don't skip tests** - Tests are mandatory
2. **Don't ignore errors** - Always handle errors explicitly
3. **Don't expose sensitive data** - Review all outputs
4. **Don't assume requirements** - Ask clarifying questions
5. **Don't break backward compatibility** - Discuss first
6. **Don't skip validation** - Validate all inputs
7. **Don't forget documentation** - Keep docs updated
8. **Don't ignore linting** - Fix all lint errors

## When in Doubt

- **Ask questions** - Better to ask than assume
- **Write tests** - Tests clarify requirements
- **Review existing code** - Follow established patterns
- **Discuss trade-offs** - Consider alternatives
- **Start small** - Implement incrementally
- **Verify assumptions** - Test your understanding

---

**Remember**: Quality over speed. It's better to ask questions and write tests than to implement something incorrectly.
