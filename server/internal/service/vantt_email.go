package service

import "strings"

// vantt 0008: MULTICA_EMAIL_LOG_DOMAINS (comma list, e.g. "opc.local") keeps mail to
// internal bot accounts out of the real mail backend. Recipients on a listed domain
// (or a subdomain of it) get the same "[DEV] ..." stdout line as DEV mode, so their
// login codes can still be read from the backend log; everyone else is sent normally.
// Unset = upstream behaviour.

func parseLogDomains(raw string) []string {
	var out []string
	for _, d := range strings.Split(raw, ",") {
		d = strings.Trim(strings.ToLower(strings.TrimSpace(d)), ".@")
		if d != "" {
			out = append(out, d)
		}
	}
	return out
}

func (s *EmailService) logOnly(to string) bool {
	if len(s.logDomains) == 0 {
		return false
	}
	at := strings.LastIndex(to, "@")
	if at < 0 {
		return false
	}
	domain := strings.ToLower(strings.TrimSpace(to[at+1:]))
	for _, d := range s.logDomains {
		if domain == d || strings.HasSuffix(domain, "."+d) {
			return true
		}
	}
	return false
}
