package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/builtbyrobben/docusign-cli/internal/outfmt"
)

type DocumentsCmd struct {
	List     DocumentsListCmd     `cmd:"" help:"List documents in an envelope"`
	Download DocumentsDownloadCmd `cmd:"" help:"Download a document as PDF"`
}

type DocumentsListCmd struct {
	EnvelopeID string `arg:"" required:"" help:"Envelope ID"`
}

func (cmd *DocumentsListCmd) Run(ctx context.Context) error {
	client, err := getDocuSignClient(ctx)
	if err != nil {
		return err
	}

	result, err := client.ListDocuments(ctx, cmd.EnvelopeID)
	if err != nil {
		return err
	}

	if outfmt.IsJSON(ctx) {
		return outfmt.WriteJSON(os.Stdout, result)
	}
	if outfmt.IsPlain(ctx) {
		headers := []string{"DOCUMENT_ID", "NAME", "TYPE", "PAGES"}
		var rows [][]string
		for _, doc := range result.EnvelopeDocuments {
			rows = append(rows, []string{doc.DocumentID, doc.Name, doc.Type, doc.Pages})
		}
		return outfmt.WritePlain(os.Stdout, headers, rows)
	}

	if len(result.EnvelopeDocuments) == 0 {
		fmt.Fprintln(os.Stderr, "No documents found")
		return nil
	}

	fmt.Fprintf(os.Stderr, "Documents in envelope %s\n\n", result.EnvelopeID)

	for _, doc := range result.EnvelopeDocuments {
		fmt.Printf("ID:    %s\n", doc.DocumentID)
		fmt.Printf("Name:  %s\n", doc.Name)

		if doc.Type != "" {
			fmt.Printf("Type:  %s\n", doc.Type)
		}

		if doc.Pages != "" {
			fmt.Printf("Pages: %s\n", doc.Pages)
		}

		fmt.Println()
	}

	return nil
}

type DocumentsDownloadCmd struct {
	EnvelopeID string `arg:"" required:"" help:"Envelope ID"`
	DocumentID string `arg:"" required:"" help:"Document ID"`
	Output     string `help:"Output file path (default: stdout)" name:"output"`
}

func (cmd *DocumentsDownloadCmd) Run(ctx context.Context) error {
	client, err := getDocuSignClient(ctx)
	if err != nil {
		return err
	}

	data, err := client.DownloadDocument(ctx, cmd.EnvelopeID, cmd.DocumentID)
	if err != nil {
		return err
	}

	if cmd.Output != "" {
		if writeErr := os.WriteFile(cmd.Output, data, 0o600); writeErr != nil {
			return fmt.Errorf("write file: %w", writeErr)
		}

		fmt.Fprintf(os.Stderr, "Document saved to %s (%d bytes)\n", cmd.Output, len(data))

		return nil
	}

	_, err = os.Stdout.Write(data)

	return err
}
