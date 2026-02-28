package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/builtbyrobben/docusign-cli/internal/docusign"
	"github.com/builtbyrobben/docusign-cli/internal/outfmt"
)

type ViewsCmd struct {
	Signing ViewsSigningCmd `cmd:"" help:"Create an embedded signing URL"`
}

type ViewsSigningCmd struct {
	EnvelopeID   string `arg:"" required:"" help:"Envelope ID"`
	SignerEmail  string `required:"" help:"Signer's email address" name:"signer-email"`
	SignerName   string `required:"" help:"Signer's name" name:"signer-name"`
	ReturnURL    string `required:"" help:"Return URL after signing" name:"return-url"`
	ClientUserID string `help:"Client user ID for embedded signing" name:"client-user-id"`
}

func (cmd *ViewsSigningCmd) Run(ctx context.Context) error {
	client, err := getDocuSignClient(ctx)
	if err != nil {
		return err
	}

	req := docusign.RecipientViewRequest{
		ReturnURL:            cmd.ReturnURL,
		AuthenticationMethod: "none",
		Email:                cmd.SignerEmail,
		UserName:             cmd.SignerName,
		ClientUserID:         cmd.ClientUserID,
	}

	result, err := client.CreateRecipientView(ctx, cmd.EnvelopeID, req)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}
	if outfmt.IsPlain(ctx) {
		return outfmt.WritePlain(os.Stdout,
			[]string{"URL"},
			[][]string{{result.URL}},
		)
	}

	fmt.Printf("%s\n", result.URL)

	return nil
}
