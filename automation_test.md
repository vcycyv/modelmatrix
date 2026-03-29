# Automation Testing Rules for React + Golang Project

## Objective

Generate automation tests that:
- validate real system behavior
- catch bugs early (especially SQL and integration issues)
- run fast and reliably in CI
- are easy to maintain
- do not rely on manual testing

---

## Core Principles

- Test at the lowest level that provides sufficient confidence
- Prefer integration tests over E2E for backend behavior
- Avoid over-mocking
- Avoid workflow coupling
- Use fixtures/builders for data setup
- Keep tests independent
- Keep tests deterministic
- Each test verifies one behavior
- Prefer simple and clear tests
- Minimize E2E tests

---

## Test Type Selection

Use unit tests when testing:
- pure logic
- validation
- transformation
- service layer decisions

Use integration tests when testing:
- database queries
- repository layer
- GORM behavior
- REST APIs
- interactions between components

Use frontend tests when testing:
- React UI behavior
- user interaction
- component state changes

Use E2E tests only for:
- critical workflows across the system

---

## Backend Testing (Golang)

Unit tests:
- test business logic
- do not use database
- do not use network
- use mocks for dependencies

Integration tests:
- test repository and database interactions
- use real database
- test actual queries and schema behavior

---

## API Integration Tests

- call real REST endpoints
- use real database
- verify responses and data changes
- do not mock backend behavior

---

## Data Setup

Do not use application workflows to prepare test data.

Avoid:
- calling create API to prepare data for other tests

Use:
- direct database setup
- fixtures or builders to create required state

---

## Fixtures and Builders

Fixtures:
- create data directly in the database
- provide known test state

Builders:
- allow flexible creation of test data
- improve readability and reuse

Benefits:
- faster tests
- simpler setup
- no dependency on other features
- easier debugging

---

## Workflow Coupling

Avoid:
- tests depending on other features
- chaining operations like create → edit → delete unless testing workflow

Prefer:
- setup → test → verify
- independent test execution

---

## Frontend Testing (React)

Use component tests as primary frontend tests:
- test user interactions
- test UI state changes
- mock API calls

Use UI unit tests for:
- rendering
- props behavior

Avoid using E2E tests for:
- small UI features
- simple interactions

---

## E2E Testing

Use only for critical workflows:
- create project
- import model
- run scoring
- deploy or publish

Avoid:
- testing every UI feature with E2E
- duplicating coverage already handled by other tests

---

## Over-Mocking

Avoid:
- mocking database behavior
- mocking repository in integration tests
- mocking internal system logic

Use mocks only for:
- external systems
- third-party APIs

Principle:
- mock what you do not own
- use real implementations for what you need to verify

---

## Coverage Strategy

Critical features:
- covered by integration tests and E2E tests

Non-critical features:
- covered by integration tests and frontend tests

Do not rely on E2E for full coverage

---

## Test Execution

Tests should:
- run quickly locally
- run automatically in CI
- not depend on environment instability
- not require manual preparation

---


## Final Rules for Test Generation

- choose the correct test type first
- use fixtures for data setup
- avoid unnecessary dependencies
- avoid over-mocking
- keep tests minimal and focused
- verify observable behavior
- prefer integration tests over E2E
- do not generate excessive E2E tests

---

## Final Statement

Fast tests with real behavior coverage and minimal E2E provide a scalable and reliable testing strategy