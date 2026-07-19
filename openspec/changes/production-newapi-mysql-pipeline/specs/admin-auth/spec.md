# Administrator authentication capability

## Requirements

### Requirement: authenticated administration

The system SHALL require a valid administrator session for project, workflow, provider, run, and asset APIs.

#### Scenario: unauthenticated request

- WHEN a client without a valid session accesses a protected API
- THEN the server returns HTTP 401 without exposing protected data.

#### Scenario: valid login

- WHEN configured credentials are submitted
- THEN the server issues a signed, expiring, HttpOnly, SameSite=Strict session cookie.

#### Scenario: logout

- WHEN an authenticated administrator logs out
- THEN the session cookie is expired and subsequent protected requests return HTTP 401.

### Requirement: provider secret safety

Provider credentials SHALL be encrypted at rest and SHALL never be returned by an API.

#### Scenario: update without token

- WHEN provider metadata is updated with an empty token field
- THEN the existing encrypted token is retained.

#### Scenario: wrong application secret

- WHEN provider ciphertext cannot be decrypted with `APP_SECRET_KEY`
- THEN model execution fails safely and never logs ciphertext or plaintext.
