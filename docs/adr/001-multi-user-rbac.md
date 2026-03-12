# ADR 001: Multi-User System with RBAC

## Status
Accepted

## Context
ModBridge originally only supported a single admin user. This became a limitation for:
- Team collaboration
- Security compliance (auditing, accountability)
- Operational needs (operators vs. administrators)
- Customer deployments with multiple stakeholders

## Decision
Implement a Role-Based Access Control (RBAC) system with:
- Four predefined roles: Admin, Operator, Viewer, Auditor
- User authentication with local password storage
- Session-based authentication
- Permission checks on all API endpoints
- LDAP/Active Directory integration for enterprise deployments

## Consequences
**Positive:**
- Better security and accountability
- Support for team collaboration
- Meets enterprise compliance requirements
- Flexible permission model

**Negative:**
- Increased complexity
- Additional database tables required
- More UI components needed
- Migration required for existing deployments

## Alternatives Considered
1. **Single user with API keys** - Too limiting for enterprise use
2. **OAuth2/OIDC only** - Overkill for small deployments
3. **No authentication** - Unacceptable for production use

## Implementation
- Database tables: `users`, `user_sessions`
- Package: `pkg/users`, `pkg/rbac`
- Frontend: User management UI
- API: `/api/users/*`, `/api/auth/*`

## References
- [NIST RBAC](https://csrc.nist.gov/projects/role-based-access-control)
- [OWASP Authentication](https://owasp.org/www-project/application-security-verification-standard/)
