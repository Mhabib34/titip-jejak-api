package service

import (
	"context"
	"fmt"
	"titip-jejak-api/internal/model"

	"github.com/resend/resend-go/v2"
)

// EmailService mengirim email notifikasi via Resend.
type EmailService struct {
	client    *resend.Client
	fromEmail string // contoh: "TemuKan <noreply@temukan.id>"
	appURL    string // contoh: "https://temukan.id"
}

// NewEmailService membuat instance EmailService baru.
// apiKey  = RESEND_API_KEY dari environment.
// from    = alamat pengirim terverifikasi di Resend.
// appURL  = base URL aplikasi untuk link di email.
func NewEmailService(apiKey, from, appURL string) *EmailService {
	return &EmailService{
		client:    resend.NewClient(apiKey),
		fromEmail: from,
		appURL:    appURL,
	}
}

// MatchEmailPayload berisi semua data yang dibutuhkan untuk render email match.
type MatchEmailPayload struct {
	Match         *model.Match
	FoundReport   model.Report
	MissingReport model.Report
	RecipientUser model.User // user penerima email
	Role          string     // "finder" atau "seeker"
}

// SendMatchNotification mengirim email ke satu penerima (finder atau seeker).
func (s *EmailService) SendMatchNotification(ctx context.Context, payload MatchEmailPayload) error {
	subject, body := s.buildEmail(payload)

	params := &resend.SendEmailRequest{
		From:    s.fromEmail,
		To:      []string{payload.RecipientUser.Email},
		Subject: subject,
		Html:    body,
	}

	_, err := s.client.Emails.SendWithContext(ctx, params)
	if err != nil {
		return fmt.Errorf("resend: failed to send to %s: %w", payload.RecipientUser.Email, err)
	}
	return nil
}

// buildEmail membuat subject dan HTML body berdasarkan role penerima.
func (s *EmailService) buildEmail(p MatchEmailPayload) (subject, htmlBody string) {
	matchURL := fmt.Sprintf("%s/matches/%s", s.appURL, p.Match.ID)

	if p.Role == "finder" {
		subject = fmt.Sprintf("[TemuKan] Laporan Anda cocok dengan laporan kehilangan (skor: %d/100)", p.Match.Score)
		htmlBody = s.finderEmailHTML(p, matchURL)
	} else {
		subject = fmt.Sprintf("[TemuKan] Ditemukan kemungkinan kecocokan untuk orang yang Anda cari (skor: %d/100)", p.Match.Score)
		htmlBody = s.seekerEmailHTML(p, matchURL)
	}
	return
}

func (s *EmailService) finderEmailHTML(p MatchEmailPayload, matchURL string) string {
	personName := "seseorang"
	if p.MissingReport.Name != nil && *p.MissingReport.Name != "" {
		personName = *p.MissingReport.Name
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;padding:0;background:#f4f4f5;font-family:Arial,sans-serif;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background:#f4f4f5;padding:32px 0;">
    <tr><td align="center">
      <table width="600" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:8px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,.08);">

        <!-- Header -->
        <tr><td style="background:#16a34a;padding:24px 32px;">
          <h1 style="margin:0;color:#ffffff;font-size:22px;">🔍 TemuKan</h1>
        </td></tr>

        <!-- Body -->
        <tr><td style="padding:32px;">
          <h2 style="margin:0 0 8px;color:#111827;font-size:18px;">Halo, %s!</h2>
          <p style="margin:0 0 16px;color:#374151;font-size:15px;line-height:1.6;">
            Laporan penemuan Anda <strong>mungkin cocok</strong> dengan laporan kehilangan
            <strong>%s</strong> di <strong>%s</strong>.
          </p>

          <!-- Score badge -->
          <table cellpadding="0" cellspacing="0" style="margin:0 0 24px;">
            <tr><td style="background:#f0fdf4;border:1px solid #bbf7d0;border-radius:6px;padding:12px 20px;text-align:center;">
              <span style="font-size:28px;font-weight:bold;color:#16a34a;">%d</span>
              <span style="font-size:14px;color:#16a34a;">/100</span>
              <p style="margin:4px 0 0;font-size:12px;color:#6b7280;">Skor Kecocokan</p>
            </td></tr>
          </table>

          <!-- Score breakdown -->
          <table width="100%%" cellpadding="8" cellspacing="0" style="background:#f9fafb;border-radius:6px;margin-bottom:24px;font-size:13px;color:#374151;">
            <tr style="border-bottom:1px solid #e5e7eb;">
              <td>📍 Lokasi</td><td align="right">maks. 40 poin</td>
            </tr>
            <tr style="border-bottom:1px solid #e5e7eb;">
              <td>⚥ Jenis kelamin</td><td align="right">maks. 30 poin</td>
            </tr>
            <tr style="border-bottom:1px solid #e5e7eb;">
              <td>🎂 Perkiraan usia</td><td align="right">maks. 20 poin</td>
            </tr>
            <tr>
              <td>📝 Deskripsi fisik</td><td align="right">maks. 10 poin</td>
            </tr>
          </table>

          <p style="margin:0 0 24px;color:#374151;font-size:14px;">
            Silakan buka halaman <strong>Matches</strong> untuk melihat detail dan menghubungi pihak keluarga.
          </p>

          <a href="%s" style="display:inline-block;background:#16a34a;color:#ffffff;text-decoration:none;padding:12px 28px;border-radius:6px;font-weight:bold;font-size:15px;">
            Lihat Detail Kecocokan →
          </a>
        </td></tr>

        <!-- Footer -->
        <tr><td style="background:#f9fafb;padding:16px 32px;border-top:1px solid #e5e7eb;">
          <p style="margin:0;font-size:12px;color:#9ca3af;text-align:center;">
            Email ini dikirim otomatis oleh sistem TemuKan. Jangan balas email ini.<br>
            © 2025 TemuKan — Menghubungkan penemu, keluarga, dan relawan di Indonesia.
          </p>
        </td></tr>

      </table>
    </td></tr>
  </table>
</body>
</html>`,
		p.RecipientUser.Name,
		personName,
		p.MissingReport.City,
		p.Match.Score,
		matchURL,
	)
}

func (s *EmailService) seekerEmailHTML(p MatchEmailPayload, matchURL string) string {
	personName := "orang yang Anda cari"
	if p.MissingReport.Name != nil && *p.MissingReport.Name != "" {
		personName = *p.MissingReport.Name
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="margin:0;padding:0;background:#f4f4f5;font-family:Arial,sans-serif;">
  <table width="100%%" cellpadding="0" cellspacing="0" style="background:#f4f4f5;padding:32px 0;">
    <tr><td align="center">
      <table width="600" cellpadding="0" cellspacing="0" style="background:#ffffff;border-radius:8px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,.08);">

        <!-- Header -->
        <tr><td style="background:#2563eb;padding:24px 32px;">
          <h1 style="margin:0;color:#ffffff;font-size:22px;">🔍 TemuKan</h1>
        </td></tr>

        <!-- Body -->
        <tr><td style="padding:32px;">
          <h2 style="margin:0 0 8px;color:#111827;font-size:18px;">Halo, %s!</h2>
          <p style="margin:0 0 16px;color:#374151;font-size:15px;line-height:1.6;">
            Ada kabar baik! Sistem kami menemukan laporan yang <strong>mungkin cocok</strong>
            dengan <strong>%s</strong> yang Anda cari di <strong>%s</strong>.
          </p>

          <!-- Score badge -->
          <table cellpadding="0" cellspacing="0" style="margin:0 0 24px;">
            <tr><td style="background:#eff6ff;border:1px solid #bfdbfe;border-radius:6px;padding:12px 20px;text-align:center;">
              <span style="font-size:28px;font-weight:bold;color:#2563eb;">%d</span>
              <span style="font-size:14px;color:#2563eb;">/100</span>
              <p style="margin:4px 0 0;font-size:12px;color:#6b7280;">Skor Kecocokan</p>
            </td></tr>
          </table>

          <p style="margin:0 0 8px;color:#374151;font-size:14px;">Detail laporan yang ditemukan:</p>
          <table width="100%%" cellpadding="8" cellspacing="0" style="background:#f9fafb;border-radius:6px;margin-bottom:24px;font-size:13px;color:#374151;">
            <tr style="border-bottom:1px solid #e5e7eb;">
              <td style="color:#6b7280;">Lokasi ditemukan</td>
              <td align="right"><strong>%s</strong></td>
            </tr>
            <tr style="border-bottom:1px solid #e5e7eb;">
              <td style="color:#6b7280;">Provinsi</td>
              <td align="right">%s</td>
            </tr>
          </table>

          <p style="margin:0 0 24px;color:#374151;font-size:14px;">
            Buka halaman <strong>Matches</strong> untuk melihat detail lengkap dan menghubungi penemu.
          </p>

          <a href="%s" style="display:inline-block;background:#2563eb;color:#ffffff;text-decoration:none;padding:12px 28px;border-radius:6px;font-weight:bold;font-size:15px;">
            Lihat Detail Kecocokan →
          </a>
        </td></tr>

        <!-- Footer -->
        <tr><td style="background:#f9fafb;padding:16px 32px;border-top:1px solid #e5e7eb;">
          <p style="margin:0;font-size:12px;color:#9ca3af;text-align:center;">
            Email ini dikirim otomatis oleh sistem TemuKan. Jangan balas email ini.<br>
            © 2025 TemuKan — Menghubungkan penemu, keluarga, dan relawan di Indonesia.
          </p>
        </td></tr>

      </table>
    </td></tr>
  </table>
</body>
</html>`,
		p.RecipientUser.Name,
		personName,
		p.FoundReport.City,
		p.Match.Score,
		p.FoundReport.City,
		p.FoundReport.Province,
		matchURL,
	)
}
