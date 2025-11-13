# Security Incident Report: GCP Service Account Key Exposure

**Incident ID**: SEC-2025-01
**Date Detected**: 2025-01-13
**Severity**: **HIGH**
**Status**: Remediated
**Reporter**: Google Cloud Security Team

---

## Executive Summary

A Google Cloud Platform (GCP) service account private key was accidentally committed to the public GitHub repository `justin4957/ec2-test-apps`. The exposed credential provided access to Vertex AI Imagen API services. Google detected the exposure and disabled the key. This report documents the incident, impact assessment, and remediation steps taken.

---

## Incident Details

### Exposed Credential

- **Service Account**: `vertex-ai-imagen@notspies.iam.gserviceaccount.com`
- **Key ID**: `17dea7254b9798ab3f31276e546dc6aaa5aa2a29`
- **File Path**: `error-generator/gcp-service-account.json`
- **Exposed Commit**: `95950fbe98ccf47e86de99be0ce8c716e7793be9`
- **Public URL**: `https://github.com/justin4957/ec2-test-apps/blob/95950fbe98ccf47e86de99be0ce8c716e7793be9/error-generator/gcp-service-account.json`
- **Exposure Duration**: Unknown start date - 2025-01-13 (detection date)

### Google Cloud Detection Notice

```
Dear Developer,
We detected and will disable a publicly exposed Service Account authentication
credential associated with the following Google Cloud Platform account:

vertex-ai-imagen@notspies.iam.gserviceaccount.com with key ID
17dea7254b9798ab3f31276e546dc6aaa5aa2a29

This key was found at the following URL:
https://github.com/justin4957/ec2-test-apps/blob/95950fbe98ccf47e86de99be0ce8c716e7793be9/error-generator/gcp-service-account.json
```

---

## Impact Assessment

### Service Account Permissions

The exposed service account had access to:
- **Vertex AI Imagen API** - Image generation service
- **Project**: `notspies`
- **Location**: `us-central1`

### Potential Impact

**HIGH RISK SCENARIOS:**
- ✅ **Unauthorized API Usage**: Attacker could generate images using Vertex AI at our expense
- ✅ **Cost Escalation**: Malicious actors could rack up significant GCP charges
- ✅ **Service Quota Exhaustion**: Could DoS the image generation service by exhausting quotas
- ⚠️ **Data Access**: Limited - service account was scoped only to Vertex AI, not storage/compute

**MITIGATED BY:**
- Google's automatic key disabling upon detection
- Limited scope of service account (Vertex AI only, not full project access)
- No evidence of unauthorized usage detected in GCP audit logs (to be verified)

### Services Affected

- **error-generator** application - Meme generation feature
- Production deployment at `https://notspies.org`

---

## Root Cause Analysis

### How It Happened

1. **Developer error**: GCP service account JSON file was created in `error-generator/` directory
2. **Insufficient .gitignore rules**: `.gitignore` did not exclude `*service-account*.json` files
3. **Code committed with credentials**: File was committed in commit `95950fb` ("stuffs")
4. **Pushed to public repository**: Credentials became publicly accessible on GitHub
5. **Google automated detection**: Google's credential scanning detected the exposure

### Contributing Factors

- No pre-commit hooks to detect credentials
- Service account credentials stored as file instead of environment variable
- Dockerfile copied the service account file into Docker image
- No secrets scanning CI/CD pipeline
- Lack of developer training on credential management

---

## Remediation Actions Taken

### Immediate Actions (Completed)

1. ✅ **Key Disabled**: Google automatically disabled the exposed key
2. ✅ **Repository Access Review**: Confirmed repository is public (risk confirmed)
3. ✅ **Code Changes Implemented**:
   - Removed `error-generator/gcp-service-account.json` from repository
   - Updated `.gitignore` to exclude all GCP service account files
   - Modified `meme_generator.go` to use `GCP_SERVICE_ACCOUNT_JSON` environment variable
   - Updated `Dockerfile` to no longer copy service account file
4. ✅ **Documentation Created**: This security incident report

### Pending Actions (To Be Completed)

- [ ] **Generate New Service Account Key**: Create replacement key via GCP Console
- [ ] **Update Deployment Configuration**: Add `GCP_SERVICE_ACCOUNT_JSON` to `.env.ec2`
- [ ] **Rebuild and Redeploy**: Build new Docker image and deploy to EC2
- [ ] **Verify Functionality**: Test meme generation feature in production
- [ ] **Git History Cleanup**: Consider using BFG Repo-Cleaner or git-filter-repo to remove credential from history
- [ ] **Review GCP Audit Logs**: Check for unauthorized API calls during exposure window
- [ ] **Review Other Credentials**: Audit `.env.ec2` for other potentially exposed secrets

---

## Long-Term Prevention Measures

### Technical Controls

1. **Pre-commit Hooks**:
   - Install `git-secrets` or `detect-secrets` to scan for credentials
   - Configure to reject commits containing patterns like:
     - `private_key`
     - `service_account`
     - `BEGIN PRIVATE KEY`

2. **CI/CD Secrets Scanning**:
   - Add GitHub Actions workflow with `truffleHog` or `gitleaks`
   - Scan pull requests for credentials before merge
   - Block merges if secrets detected

3. **Secret Management**:
   - Migrate all secrets to AWS Secrets Manager or HashiCorp Vault
   - Use IAM roles for service authentication where possible
   - Never store credentials as files in application directories

4. **Repository Protections**:
   - Enable GitHub secret scanning (already automatic for public repos)
   - Consider making repository private
   - Enable branch protection rules requiring reviews

### Process Controls

1. **Developer Training**:
   - Conduct security awareness training on credential management
   - Document secure credential practices in `CLAUDE.md`
   - Include credential handling in onboarding checklist

2. **Code Review Guidelines**:
   - Require explicit reviewer check for `.env` and credential changes
   - Mandate review of all `Dockerfile` changes
   - Flag any `*.json` files in sensitive directories

3. **Incident Response Plan**:
   - Document credential exposure response playbook
   - Define escalation path for security incidents
   - Establish SLA for credential rotation (e.g., 24 hours)

---

## Lessons Learned

### What Went Well

- Google's automated detection worked quickly
- Key was disabled before confirmed abuse
- Scoped service account limited blast radius
- Clear notification from Google included exact location

### What Could Be Improved

- Need automated credential scanning in development workflow
- `.gitignore` should have been more comprehensive from the start
- Should have used environment variables for all secrets from day one
- No monitoring/alerting for unusual GCP API activity

### Action Items

1. Implement pre-commit hooks for secret detection (Priority: HIGH)
2. Add CI/CD secrets scanning (Priority: HIGH)
3. Migrate all credentials to secrets manager (Priority: MEDIUM)
4. Conduct security audit of all repositories (Priority: MEDIUM)
5. Document credential management standards (Priority: HIGH)

---

## Timeline

| Time | Event |
|------|-------|
| Unknown | Service account key committed to repository in commit `95950fb` |
| Unknown | Repository remained public with exposed credentials |
| 2025-01-13 | Google Cloud Security detected exposed key |
| 2025-01-13 | Google sent notification email to developer |
| 2025-01-13 | Google automatically disabled the exposed key |
| 2025-01-13 | Security incident investigation initiated |
| 2025-01-13 | Remediation code changes implemented |
| 2025-01-13 | Security incident report created |
| Pending | New key generation and deployment |

---

## Additional Notes

### Related Files Modified

- `error-generator/gcp-service-account.json` (DELETED)
- `.gitignore` (UPDATED - added GCP credential patterns)
- `error-generator/meme_generator.go` (UPDATED - use env var)
- `error-generator/Dockerfile` (UPDATED - removed credential copy)

### Environment Variables Required

New environment variable required in `.env.ec2`:

```bash
# GCP Service Account JSON (base64 encoded or inline JSON string)
# SECURITY: Generate new key from GCP Console after incident
GCP_SERVICE_ACCOUNT_JSON='{"type":"service_account","project_id":"notspies",...}'
```

### References

- Google Cloud IAM Best Practices: https://cloud.google.com/iam/docs/best-practices-for-managing-service-account-keys
- GitHub Secret Scanning: https://docs.github.com/en/code-security/secret-scanning
- git-secrets: https://github.com/awslabs/git-secrets
- BFG Repo-Cleaner: https://rtyley.github.io/bfg-repo-cleaner/

---

## Approval

**Prepared by**: Claude Code (AI Assistant)
**Date**: 2025-01-13
**Review Required**: Engineering Lead, Security Team

---

## Appendix: Code Changes Summary

### Before (Insecure)

```go
// Read service account JSON from file
credsJSON, err := os.ReadFile("gcp-service-account.json")
```

```dockerfile
COPY gcp-service-account.json ./
```

### After (Secure)

```go
// Read service account JSON from environment variable
credsJSON := os.Getenv("GCP_SERVICE_ACCOUNT_JSON")
if credsJSON == "" {
    return "", fmt.Errorf("GCP_SERVICE_ACCOUNT_JSON environment variable not set")
}
```

```dockerfile
# No longer copying credential file - using environment variable
```

---

**END OF REPORT**
