# ADR 002: Comprehensive Audit Logging

## Status
Accepted

## Context
Regulatory compliance and security best practices require:
- Complete audit trail of all system changes
- Ability to investigate security incidents
- Accountability for user actions
- Compliance with SOC2, ISO27001, GDPR

## Decision
Implement comprehensive audit logging covering:
- All user authentication events
- Configuration changes
- Proxy management actions
- System operations
- Failed access attempts

## Consequences
**Positive:**
- Security incident investigation capability
- Regulatory compliance
- Accountability and non-repudiation
- Operational insights

**Negative:**
- Storage requirements for log data
- Performance overhead (minimal with buffering)
- Privacy considerations (GDPR)

## Implementation
- Database table: `audit_log`
- Package: `pkg/audit`
- Async logging with buffered channel
- Export functionality (JSON, CSV)
- Retention policies

## Retention Policy
- Online: 90 days in SQLite database
- Archive: Export to external storage for long-term retention
- Privacy: Anonymize old logs per GDPR requirements
