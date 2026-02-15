package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/builtbyrobben/docusign-cli/internal/outfmt"
)

type RecipientsCmd struct {
	List RecipientsListCmd `cmd:"" help:"List recipients of an envelope"`
}

type RecipientsListCmd struct {
	EnvelopeID string `arg:"" required:"" help:"Envelope ID"`
}

func (cmd *RecipientsListCmd) Run(ctx context.Context) error {
	client, err := getDocuSignClient(ctx)
	if err != nil {
		return err
	}

	result, err := client.ListRecipients(ctx, cmd.EnvelopeID)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}

	if len(result.Signers) == 0 {
		fmt.Fprintln(os.Stderr, "No signers found")
		return nil
	}

	fmt.Fprintf(os.Stderr, "Recipients for envelope %s\n\n", cmd.EnvelopeID)

	for _, signer := range result.Signers {
		fmt.Printf("Signer: %s <%s>\n", signer.Name, signer.Email)
		fmt.Printf("  ID:     %s\n", signer.RecipientID)

		if signer.Status != "" {
			fmt.Printf("  Status: %s\n", signer.Status)
		}

		if signer.RoutingOrder != "" {
			fmt.Printf("  Order:  %s\n", signer.RoutingOrder)
		}

		if signer.SignedAt != "" {
			fmt.Printf("  Signed: %s\n", signer.SignedAt)
		}

		fmt.Println()
	}

	return nil
}
