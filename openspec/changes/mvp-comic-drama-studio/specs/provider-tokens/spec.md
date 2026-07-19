# Provider token capability

## Requirements

### Requirement: secret-safe provider configuration

The system SHALL persist provider credentials encrypted and SHALL never return plaintext credentials from list or detail endpoints.

#### Scenario: list providers

- WHEN a provider with a credential is listed
- THEN only a masked credential hint and routing metadata are returned.

### Requirement: provider lifecycle

The system SHALL allow a provider to be created, enabled, disabled and deleted.

