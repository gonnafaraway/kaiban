package integration

// Integration kinds accepted by HealthProbeURL; they mirror domain/integration.Type.
const (
	KindJira       = "jira"
	KindConfluence = "confluence"
	KindGitLab     = "gitlab"
	KindGitHub     = "github"
)

// HealthProbeURL returns the "who am I" endpoint for a connectivity test.
// A bare base URL is useless here: it usually answers 200 for anonymous users,
// so credentials would look valid even when they are not.
func HealthProbeURL(kind, baseURL string) string {
	base := apiBase(baseURL)
	if base == "" {
		return ""
	}
	switch kind {
	case KindJira:
		return base + "/rest/api/2/myself"
	case KindConfluence:
		return base + "/rest/api/user/current"
	case KindGitLab:
		return base + "/api/v4/user"
	case KindGitHub:
		return NewGitHubClient(base, "").BaseURL + "/user"
	default:
		return base
	}
}

// HealthProbeAuth returns request headers for a probe of the given kind.
func HealthProbeAuth(kind, email, token string) map[string]string {
	switch kind {
	case KindJira, KindConfluence:
		if email == "" && token == "" {
			return nil
		}
		return AtlassianAuth(email, token)
	case KindGitLab:
		if token == "" {
			return nil
		}
		return GitLabAuth(token)
	case KindGitHub:
		if token == "" {
			return nil
		}
		return GitHubAuth(token)
	default:
		return nil
	}
}
