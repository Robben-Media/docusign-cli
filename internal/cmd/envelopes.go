package cmd

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"github.com/builtbyrobben/docusign-cli/internal/docusign"
	"github.com/builtbyrobben/docusign-cli/internal/outfmt"
)

type EnvelopesCmd struct {
	List   EnvelopesListCmd   `cmd:"" help:"List envelopes"`
	Get    EnvelopesGetCmd    `cmd:"" help:"Get envelope details"`
	Create EnvelopesCreateCmd `cmd:"" help:"Create a new envelope"`
	Send   EnvelopesSendCmd   `cmd:"" help:"Send a draft envelope"`
	Void   EnvelopesVoidCmd   `cmd:"" help:"Void an envelope"`
	Audit  EnvelopesAuditCmd  `cmd:"" help:"Get envelope audit trail"`
}

type EnvelopesListCmd struct {
	From   string `help:"From date (ISO 8601, e.g. 2024-01-01)" name:"from"`
	Status string `help:"Filter by status (e.g. sent, completed, voided)"`
	Count  int    `help:"Number of results to return" default:"10"`
}

func (cmd *EnvelopesListCmd) Run(ctx context.Context) error {
	client, err := getDocuSignClient(ctx)
	if err != nil {
		return err
	}

	result, err := client.ListEnvelopes(ctx, cmd.From, cmd.Status, cmd.Count)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	if len(result.Envelopes) == 0 {
		fmt.Fprintln(os.Stderr, "No envelopes found")
		return nil
	}

	fmt.Fprintf(os.Stderr, "Showing %s of %s envelopes\n\n", result.ResultSetSize, result.TotalSetSize)

	for _, env := range result.Envelopes {
		fmt.Printf("ID:      %s\n", env.EnvelopeID)
		fmt.Printf("Subject: %s\n", env.EmailSubject)
		fmt.Printf("Status:  %s\n", env.Status)

		if env.SentDateTime != "" {
			fmt.Printf("Sent:    %s\n", env.SentDateTime)
		}

		fmt.Println()
	}

	return nil
}

type EnvelopesGetCmd struct {
	ID string `arg:"" required:"" help:"Envelope ID"`
}

func (cmd *EnvelopesGetCmd) Run(ctx context.Context) error {
	client, err := getDocuSignClient(ctx)
	if err != nil {
		return err
	}

	result, err := client.GetEnvelope(ctx, cmd.ID)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	fmt.Printf("ID:      %s\n", result.EnvelopeID)
	fmt.Printf("Subject: %s\n", result.EmailSubject)
	fmt.Printf("Status:  %s\n", result.Status)

	if result.EmailBlurb != "" {
		fmt.Printf("Message: %s\n", result.EmailBlurb)
	}

	if result.CreatedAt != "" {
		fmt.Printf("Created: %s\n", result.CreatedAt)
	}

	if result.SentDateTime != "" {
		fmt.Printf("Sent:    %s\n", result.SentDateTime)
	}

	return nil
}

type EnvelopesCreateCmd struct {
	Subject     string `required:"" help:"Email subject for the envelope"`
	SignerEmail string `required:"" help:"Signer's email address" name:"signer-email"`
	SignerName  string `required:"" help:"Signer's name" name:"signer-name"`
	Document    string `required:"" help:"Path to document file"`
	Status      string `help:"Envelope status: sent or created (draft)" default:"sent"`
}

func (cmd *EnvelopesCreateCmd) Run(ctx context.Context) error {
	client, err := getDocuSignClient(ctx)
	if err != nil {
		return err
	}

	// Check file size before reading (DocuSign limit is 25MB)
	const maxDocSize = 25 * 1024 * 1024
	fi, err := os.Stat(cmd.Document)
	if err != nil {
		return fmt.Errorf("stat document file: %w", err)
	}
	if fi.Size() > maxDocSize {
		return fmt.Errorf("document file is %d bytes, exceeds DocuSign 25MB limit", fi.Size())
	}

	docBytes, err := os.ReadFile(cmd.Document)
	if err != nil {
		return fmt.Errorf("read document file: %w", err)
	}

	docBase64 := base64.StdEncoding.EncodeToString(docBytes)
	ext := filepath.Ext(cmd.Document)

	if len(ext) > 0 {
		ext = ext[1:] // remove leading dot
	}

	req := docusign.CreateEnvelopeRequest{
		EmailSubject: cmd.Subject,
		Status:       cmd.Status,
		Documents: []docusign.Document{
			{
				DocumentBase64: docBase64,
				Name:           filepath.Base(cmd.Document),
				FileExtension:  ext,
				DocumentID:     "1",
			},
		},
		Recipients: &docusign.Recipients{
			Signers: []docusign.Signer{
				{
					Email:       cmd.SignerEmail,
					Name:        cmd.SignerName,
					RecipientID: "1",
				},
			},
		},
	}

	result, err := client.CreateEnvelope(ctx, req)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	fmt.Fprintf(os.Stderr, "Envelope created\n\n")
	fmt.Printf("ID:     %s\n", result.EnvelopeID)
	fmt.Printf("Status: %s\n", result.Status)

	return nil
}

type EnvelopesSendCmd struct {
	ID string `arg:"" required:"" help:"Envelope ID"`
}

func (cmd *EnvelopesSendCmd) Run(ctx context.Context) error {
	client, err := getDocuSignClient(ctx)
	if err != nil {
		return err
	}

	if err := client.SendEnvelope(ctx, cmd.ID); err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, map[string]string{
			"status":      "success",
			"envelope_id": cmd.ID,
		})
	}

	fmt.Fprintf(os.Stderr, "Envelope %s sent\n", cmd.ID)

	return nil
}

type EnvelopesVoidCmd struct {
	ID     string `arg:"" required:"" help:"Envelope ID"`
	Reason string `required:"" help:"Reason for voiding"`
}

func (cmd *EnvelopesVoidCmd) Run(ctx context.Context) error {
	client, err := getDocuSignClient(ctx)
	if err != nil {
		return err
	}

	if err := client.VoidEnvelope(ctx, cmd.ID, cmd.Reason); err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, map[string]string{
			"status":      "success",
			"envelope_id": cmd.ID,
		})
	}

	fmt.Fprintf(os.Stderr, "Envelope %s voided\n", cmd.ID)

	return nil
}

type EnvelopesAuditCmd struct {
	ID string `arg:"" required:"" help:"Envelope ID"`
}

func (cmd *EnvelopesAuditCmd) Run(ctx context.Context) error {
	client, err := getDocuSignClient(ctx)
	if err != nil {
		return err
	}

	result, err := client.GetAuditEvents(ctx, cmd.ID)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	if len(result.AuditEvents) == 0 {
		fmt.Fprintln(os.Stderr, "No audit events found")
		return nil
	}

	fmt.Fprintf(os.Stderr, "Audit trail for envelope %s\n\n", cmd.ID)

	for i, event := range result.AuditEvents {
		fmt.Printf("Event %d:\n", i+1)

		for _, field := range event.EventFields {
			fmt.Printf("  %s: %s\n", field.Name, field.Value)
		}

		fmt.Println()
	}

	return nil
}
