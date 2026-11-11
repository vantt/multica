package service

import (
	"testing"

	"github.com/resend/resend-go/v2"
)

func TestVanttLogOnlyDomains(t *testing.T) {
	s := &EmailService{logDomains: parseLogDomains(" OPC.local, @corp.internal ,, ")}
	cases := map[string]bool{
		"agents-acme@opc.local":   true,
		"Ops-Bot@OPC.LOCAL":       true,
		"x@sub.opc.local":         true,
		"x@corp.internal":         true,
		"owner@example.com":       false,
		"x@notopc.local":          false,
		"x@opc.local.example.com": false,
		"no-at-sign":              false,
	}
	for to, want := range cases {
		if got := s.logOnly(to); got != want {
			t.Errorf("logOnly(%q) = %v, want %v", to, got, want)
		}
	}
	if (&EmailService{}).logOnly("x@opc.local") {
		t.Error("no domains configured must never be log-only")
	}
}

// A log-only recipient never reaches the mail backend: with a Resend client that
// points nowhere and an SMTP host that does not exist, both sends still succeed.
func TestVanttLogOnlySkipsBackends(t *testing.T) {
	for _, s := range []*EmailService{
		{client: resend.NewClient("re_test_invalid"), logDomains: []string{"opc.local"}},
		{smtpHost: "smtp.invalid", smtpPort: "25", logDomains: []string{"opc.local"}},
	} {
		if err := s.SendVerificationCode("agents-acme@opc.local", "123456"); err != nil {
			t.Fatalf("SendVerificationCode: %v", err)
		}
		if err := s.SendInvitationEmail("agents-acme@opc.local", "Owner", "WS", "inv-1"); err != nil {
			t.Fatalf("SendInvitationEmail: %v", err)
		}
	}
}
